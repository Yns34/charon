// Copyright © 2022 Obol Labs Inc.
//
// This program is free software: you can redistribute it and/or modify it
// under the terms of the GNU General Public License as published by the Free
// Software Foundation, either version 3 of the License, or (at your option)
// any later version.
//
// This program is distributed in the hope that it will be useful, but WITHOUT
// ANY WARRANTY; without even the implied warranty of  MERCHANTABILITY or
// FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License for
// more details.
//
// You should have received a copy of the GNU General Public License along with
// this program.  If not, see <http://www.gnu.org/licenses/>.

// Package validatorapi defines validator facing API that serves the subset of
// endpoints related to distributed validation and reverse-proxies the rest to the
// upstream beacon client.
package validatorapi

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	stdlog "log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	eth2client "github.com/attestantio/go-eth2-client"
	eth2api "github.com/attestantio/go-eth2-client/api"
	eth2v1 "github.com/attestantio/go-eth2-client/api/v1"
	eth2bellatrix "github.com/attestantio/go-eth2-client/api/v1/bellatrix"
	eth2capella "github.com/attestantio/go-eth2-client/api/v1/capella"
	eth2spec "github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/capella"
	eth2p0 "github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/gorilla/mux"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/obolnetwork/charon/app/errors"
	"github.com/obolnetwork/charon/app/eth2wrap"
	"github.com/obolnetwork/charon/app/log"
	"github.com/obolnetwork/charon/app/z"
	"github.com/obolnetwork/charon/core"
	"github.com/obolnetwork/charon/eth2util/eth2exp"
	"github.com/obolnetwork/charon/tbls/tblsconv"
)

// Handler defines the request handler providing the business logic
// for the validator API router.
type Handler interface {
	eth2client.AggregateAttestationProvider
	eth2client.AggregateAttestationsSubmitter
	eth2client.AttestationDataProvider
	eth2client.AttestationsSubmitter
	eth2client.AttesterDutiesProvider
	eth2client.BeaconBlockProposalProvider
	eth2client.BeaconBlockSubmitter
	eth2exp.BeaconCommitteeSelectionAggregator
	eth2client.BlindedBeaconBlockProposalProvider
	eth2client.BlindedBeaconBlockSubmitter
	eth2client.ProposerDutiesProvider
	eth2client.SyncCommitteeContributionProvider
	eth2client.SyncCommitteeContributionsSubmitter
	eth2client.SyncCommitteeDutiesProvider
	eth2client.SyncCommitteeMessagesSubmitter
	eth2exp.SyncCommitteeSelectionAggregator
	eth2client.ValidatorsProvider
	eth2client.ValidatorRegistrationsSubmitter
	eth2client.VoluntaryExitSubmitter
	TekuProposerConfigProvider
	// Above sorted alphabetically.
}

// NewRouter returns a new validator http server router. The http router
// translates http requests related to the distributed validator to the Handler.
// All other requests are reverse-proxied to the beacon-node address.
func NewRouter(h Handler, eth2Cl eth2wrap.Client) (*mux.Router, error) {
	// Register subset of distributed validator related endpoints.
	endpoints := []struct {
		Name    string
		Path    string
		Handler handlerFunc
	}{
		{
			Name:    "attester_duties",
			Path:    "/eth/v1/validator/duties/attester/{epoch}",
			Handler: attesterDuties(h),
		},
		{
			Name:    "proposer_duties",
			Path:    "/eth/v1/validator/duties/proposer/{epoch}",
			Handler: proposerDuties(h),
		},
		{
			Name:    "sync_committee_duties",
			Path:    "/eth/v1/validator/duties/sync/{epoch}",
			Handler: syncCommitteeDuties(h),
		},
		{
			Name:    "attestation_data",
			Path:    "/eth/v1/validator/attestation_data",
			Handler: attestationData(h),
		},
		{
			Name:    "submit_attestations",
			Path:    "/eth/v1/beacon/pool/attestations",
			Handler: submitAttestations(h),
		},
		{
			Name:    "get_validators",
			Path:    "/eth/v1/beacon/states/{state_id}/validators",
			Handler: getValidators(h),
		},
		{
			Name:    "get_validator",
			Path:    "/eth/v1/beacon/states/{state_id}/validators/{validator_id}",
			Handler: getValidator(h),
		},
		{
			Name:    "propose_block",
			Path:    "/eth/v2/validator/blocks/{slot}",
			Handler: proposeBlock(h),
		},
		{
			Name:    "submit_block",
			Path:    "/eth/v1/beacon/blocks",
			Handler: submitBlock(h),
		},
		{
			Name:    "propose_blinded_block",
			Path:    "/eth/v1/validator/blinded_blocks/{slot}",
			Handler: proposeBlindedBlock(h),
		},
		{
			Name:    "submit_blinded_block",
			Path:    "/eth/v1/beacon/blinded_blocks",
			Handler: submitBlindedBlock(h),
		},
		{
			Name:    "submit_validator_registration",
			Path:    "/eth/v1/validator/register_validator",
			Handler: submitValidatorRegistrations(h),
		},
		{
			Name:    "submit_voluntary_exit",
			Path:    "/eth/v1/beacon/pool/voluntary_exits",
			Handler: submitExit(h),
		},
		{
			Name:    "teku_proposer_config",
			Path:    "/teku_proposer_config",
			Handler: tekuProposerConfig(h),
		},
		{
			Name:    "aggregate_beacon_committee_selections",
			Path:    "/eth/v1/validator/beacon_committee_selections",
			Handler: aggregateBeaconCommitteeSelections(h),
		},
		{
			Name:    "aggregate_attestation",
			Path:    "/eth/v1/validator/aggregate_attestation",
			Handler: aggregateAttestation(h),
		},
		{
			Name:    "submit_aggregate_and_proofs",
			Path:    "/eth/v1/validator/aggregate_and_proofs",
			Handler: submitAggregateAttestations(h),
		},
		{
			Name:    "submit_sync_committee_messages",
			Path:    "/eth/v1/beacon/pool/sync_committees",
			Handler: submitSyncCommitteeMessages(h),
		},
		{
			Name:    "sync_committee_contribution",
			Path:    "/eth/v1/validator/sync_committee_contribution",
			Handler: syncCommitteeContribution(h),
		},
		{
			Name:    "submit_contribution_and_proofs",
			Path:    "/eth/v1/validator/contribution_and_proofs",
			Handler: submitContributionAndProofs(h),
		},
		{
			Name:    "submit_proposal_preparations",
			Path:    "/eth/v1/validator/prepare_beacon_proposer",
			Handler: submitProposalPreparations(),
		},
		{
			Name:    "aggregate_sync_committee_selections",
			Path:    "/eth/v1/validator/sync_committee_selections",
			Handler: aggregateSyncCommitteeSelections(h),
		},
	}

	r := mux.NewRouter()
	for _, e := range endpoints {
		r.Handle(e.Path, wrap(e.Name, e.Handler))
	}

	// Everything else is proxied
	r.PathPrefix("/").Handler(proxyHandler(eth2Cl))

	return r, nil
}

// apiErr defines a validator api error that is converted to an eth2 errorResponse.
type apiError struct {
	// StatusCode is the http status code to return, defaults to 500.
	StatusCode int
	// Message is a safe human-readable message, defaults to "Internal server error".
	Message string
	// Err is the original error, returned in debug mode.
	Err error
}

func (a apiError) Error() string {
	return fmt.Sprintf("api error[status=%d,msg=%s]: %v", a.StatusCode, a.Message, a.Err)
}

// handlerFunc is a convenient handler function providing a context, parsed path parameters,
// the request body, and returning the response struct or an error.
type handlerFunc func(ctx context.Context, params map[string]string, query url.Values, body []byte) (res interface{}, err error)

// wrap adapts the handler function returning a standard http handler.
// It does tracing, metrics and response and error writing.
func wrap(endpoint string, handler handlerFunc) http.Handler {
	wrap := func(w http.ResponseWriter, r *http.Request) {
		defer observeAPILatency(endpoint)()

		ctx := r.Context()
		ctx = log.WithTopic(ctx, "vapi")
		ctx = log.WithCtx(ctx, z.Str("vapi_endpoint", endpoint))
		ctx = withCtxDuration(ctx)

		// TODO(corver): Add support for octet-stream (SSZ).
		contentType := r.Header.Get("Content-Type")
		if contentType != "" && !strings.Contains(contentType, "application/json") {
			writeError(ctx, w, endpoint, apiError{
				StatusCode: http.StatusUnsupportedMediaType,
				Message:    fmt.Sprintf("unsupported media type %s (only application/json supported)", contentType),
			})

			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeError(ctx, w, endpoint, err)
			return
		}

		res, err := handler(ctx, mux.Vars(r), r.URL.Query(), body)
		if err != nil {
			writeError(ctx, w, endpoint, err)
			return
		}

		writeResponse(ctx, w, endpoint, res)
	}

	return wrapTrace(endpoint, wrap)
}

// wrapTrace wraps the passed handler in a OpenTelemetry tracing span.
func wrapTrace(endpoint string, handler http.HandlerFunc) http.Handler {
	return otelhttp.NewHandler(handler, "core/validatorapi."+endpoint)
}

// getValidator returns a handler function for the get validators by pubkey or index endpoint.
func getValidators(p eth2client.ValidatorsProvider) handlerFunc {
	return func(ctx context.Context, params map[string]string, query url.Values, body []byte) (interface{}, error) {
		stateID := params["state_id"]

		resp, err := getValidatorsByID(ctx, p, stateID, getValidatorIDs(query)...)
		if err != nil {
			return nil, err
		} else if len(resp) == 0 {
			resp = []v1Validator{} // Return empty json array instead of null.
		}

		return validatorsResponse{Data: resp}, nil
	}
}

// getValidator returns a handler function for the get validators by pubkey or index endpoint.
func getValidator(p eth2client.ValidatorsProvider) handlerFunc {
	return func(ctx context.Context, params map[string]string, query url.Values, body []byte) (interface{}, error) {
		stateID := params["state_id"]
		id := params["validator_id"]

		vals, err := getValidatorsByID(ctx, p, stateID, id)
		if err != nil {
			return nil, err
		} else if len(vals) == 0 {
			return nil, apiError{
				StatusCode: http.StatusNotFound,
				Message:    "NotFound",
			}
		} else if len(vals) != 1 {
			return nil, errors.New("unexpected number of validators")
		}

		return validatorResponse{Data: vals[0]}, nil
	}
}

// attestationData returns a handler function for the attestation data endpoint.
func attestationData(p eth2client.AttestationDataProvider) handlerFunc {
	return func(ctx context.Context, params map[string]string, query url.Values, _ []byte) (interface{}, error) {
		slot, err := uintQuery(query, "slot")
		if err != nil {
			return nil, err
		}

		commIdx, err := uintQuery(query, "committee_index")
		if err != nil {
			return nil, err
		}

		data, err := p.AttestationData(ctx, eth2p0.Slot(slot), eth2p0.CommitteeIndex(commIdx))
		if err != nil {
			return nil, err
		}

		return struct {
			Data *eth2p0.AttestationData `json:"data"`
		}{
			Data: data,
		}, nil
	}
}

// submitAttestations returns a handler function for the attestation submitter endpoint.
func submitAttestations(p eth2client.AttestationsSubmitter) handlerFunc {
	return func(ctx context.Context, _ map[string]string, _ url.Values, body []byte) (interface{}, error) {
		var atts []*eth2p0.Attestation
		err := json.Unmarshal(body, &atts)
		if err != nil {
			return nil, errors.Wrap(err, "unmarshal attestations")
		}

		return nil, p.SubmitAttestations(ctx, atts)
	}
}

// proposerDuties returns a handler function for the proposer duty endpoint.
func proposerDuties(p eth2client.ProposerDutiesProvider) handlerFunc {
	return func(ctx context.Context, params map[string]string, _ url.Values, body []byte) (interface{}, error) {
		epoch, err := uintParam(params, "epoch")
		if err != nil {
			return nil, err
		}

		// Note the ProposerDutiesProvider interface adds some sugar to the official eth2spec.
		// ValidatorIndices aren't provided over the wire.
		data, err := p.ProposerDuties(ctx, eth2p0.Epoch(epoch), nil)
		if err != nil {
			return nil, err
		} else if len(data) == 0 { // Return empty json array instead of null
			data = []*eth2v1.ProposerDuty{}
		}

		return proposerDutiesResponse{
			ExecutionOptimistic: false,           // TODO(dhruv): Fill this properly
			DependentRoot:       stubRoot(epoch), // TODO(corver): Fill this properly
			Data:                data,
		}, nil
	}
}

// attesterDuties returns a handler function for the attester duty endpoint.
func attesterDuties(p eth2client.AttesterDutiesProvider) handlerFunc {
	return func(ctx context.Context, params map[string]string, _ url.Values, body []byte) (interface{}, error) {
		epoch, err := uintParam(params, "epoch")
		if err != nil {
			return nil, err
		}

		var req valIndexesJSON
		if err := unmarshal(body, &req); err != nil {
			return nil, err
		}

		data, err := p.AttesterDuties(ctx, eth2p0.Epoch(epoch), req)
		if err != nil {
			return nil, err
		} else if len(data) == 0 { // Return empty json array instead of null
			data = []*eth2v1.AttesterDuty{}
		}

		return attesterDutiesResponse{
			ExecutionOptimistic: false,           // TODO(dhruv): Fill this properly
			DependentRoot:       stubRoot(epoch), // TODO(corver): Fill this properly
			Data:                data,
		}, nil
	}
}

// syncCommitteeDuties returns a handler function for the sync committee duty endpoint.
func syncCommitteeDuties(p eth2client.SyncCommitteeDutiesProvider) handlerFunc {
	return func(ctx context.Context, params map[string]string, query url.Values, body []byte) (interface{}, error) {
		epoch, err := uintParam(params, "epoch")
		if err != nil {
			return nil, err
		}

		var req valIndexesJSON
		if err := unmarshal(body, &req); err != nil {
			return nil, err
		}

		data, err := p.SyncCommitteeDuties(ctx, eth2p0.Epoch(epoch), req)
		if err != nil {
			return nil, err
		} else if len(data) == 0 { // Return empty json array instead of null
			data = []*eth2v1.SyncCommitteeDuty{}
		}

		return syncCommitteeDutiesResponse{Data: data}, nil
	}
}

// syncCommitteeContribution returns a handler function for get sync committee contribution endpoint.
func syncCommitteeContribution(s eth2client.SyncCommitteeContributionProvider) handlerFunc {
	return func(ctx context.Context, params map[string]string, query url.Values, body []byte) (interface{}, error) {
		slot, err := uintQuery(query, "slot")
		if err != nil {
			return nil, err
		}

		subcommIdx, err := uintQuery(query, "subcommittee_index")
		if err != nil {
			return nil, err
		}

		var beaconBlockRoot eth2p0.Root
		err = hexQueryFixed(query, "beacon_block_root", beaconBlockRoot[:])
		if err != nil {
			return nil, err
		}

		contribution, err := s.SyncCommitteeContribution(ctx, eth2p0.Slot(slot), subcommIdx, beaconBlockRoot)
		if err != nil {
			return nil, err
		}

		return syncCommitteeContributionResponse{Data: contribution}, nil
	}
}

// submitContributionAndProofs returns a handler function for sync committee contributions submitter endpoint.
func submitContributionAndProofs(s eth2client.SyncCommitteeContributionsSubmitter) handlerFunc {
	return func(ctx context.Context, params map[string]string, query url.Values, body []byte) (interface{}, error) {
		var contributionAndProofs []*altair.SignedContributionAndProof
		err := json.Unmarshal(body, &contributionAndProofs)
		if err != nil {
			return nil, errors.Wrap(err, "unmarshal signed contribution and proofs")
		}

		return nil, s.SubmitSyncCommitteeContributions(ctx, contributionAndProofs)
	}
}

// proposeBlock receives the randao from the validator and returns the unsigned BeaconBlock.
func proposeBlock(p eth2client.BeaconBlockProposalProvider) handlerFunc {
	return func(ctx context.Context, params map[string]string, query url.Values, body []byte) (interface{}, error) {
		slot, err := uintParam(params, "slot")
		if err != nil {
			return nil, err
		}

		var randao eth2p0.BLSSignature
		if err = hexQueryFixed(query, "randao_reveal", randao[:]); err != nil {
			return nil, err
		}

		graffiti, _, err := hexQuery(query, "graffiti") // Graffiti is optional.
		if err != nil {
			return nil, err
		}

		block, err := p.BeaconBlockProposal(ctx, eth2p0.Slot(slot), randao, graffiti)
		if err != nil {
			return nil, err
		}

		switch block.Version {
		case eth2spec.DataVersionPhase0:
			if block.Phase0 == nil {
				return 0, errors.New("no phase0 block")
			}

			return proposeBlockResponsePhase0{
				Version: "PHASE0",
				Data:    block.Phase0,
			}, nil
		case eth2spec.DataVersionAltair:
			if block.Altair == nil {
				return 0, errors.New("no altair block")
			}

			return proposeBlockResponseAltair{
				Version: "ALTAIR",
				Data:    block.Altair,
			}, nil
		case eth2spec.DataVersionBellatrix:
			if block.Bellatrix == nil {
				return 0, errors.New("no bellatrix block")
			}

			return proposeBlockResponseBellatrix{
				Version: "BELLATRIX",
				Data:    block.Bellatrix,
			}, nil
		case eth2spec.DataVersionCapella:
			if block.Capella == nil {
				return 0, errors.New("no capella block")
			}

			return proposeBlockResponseCapella{
				Version: "CAPELLA",
				Data:    block.Capella,
			}, nil
		default:
			return 0, errors.New("invalid block")
		}
	}
}

// proposeBlindedBlock receives the randao from the validator and returns the unsigned BlindedBeaconBlock.
func proposeBlindedBlock(p eth2client.BlindedBeaconBlockProposalProvider) handlerFunc {
	return func(ctx context.Context, params map[string]string, query url.Values, body []byte) (interface{}, error) {
		slot, err := uintParam(params, "slot")
		if err != nil {
			return nil, err
		}

		var randao eth2p0.BLSSignature
		if err := hexQueryFixed(query, "randao_reveal", randao[:]); err != nil {
			return nil, err
		}

		block, err := p.BlindedBeaconBlockProposal(ctx, eth2p0.Slot(slot), randao, nil)
		if err != nil {
			return nil, err
		}

		switch block.Version {
		case eth2spec.DataVersionBellatrix:
			if block.Bellatrix == nil {
				return 0, errors.New("no bellatrix block")
			}

			return proposeBlindedBlockResponseBellatrix{
				Version: "BELLATRIX",
				Data:    block.Bellatrix,
			}, nil
		case eth2spec.DataVersionCapella:
			if block.Capella == nil {
				return 0, errors.New("no capella block")
			}

			return proposeBlindedBlockResponseCapella{
				Version: "CAPELLA",
				Data:    block.Capella,
			}, nil
		default:
			return 0, errors.New("invalid block")
		}
	}
}

func submitBlock(p eth2client.BeaconBlockSubmitter) handlerFunc {
	return func(ctx context.Context, _ map[string]string, _ url.Values, body []byte) (interface{}, error) {
		capellaBlock := new(capella.SignedBeaconBlock)
		err := capellaBlock.UnmarshalJSON(body)
		if err == nil {
			block := &eth2spec.VersionedSignedBeaconBlock{
				Version: eth2spec.DataVersionCapella,
				Capella: capellaBlock,
			}

			return nil, p.SubmitBeaconBlock(ctx, block)
		}

		bellatrixBlock := new(bellatrix.SignedBeaconBlock)
		err = bellatrixBlock.UnmarshalJSON(body)
		if err == nil {
			block := &eth2spec.VersionedSignedBeaconBlock{
				Version:   eth2spec.DataVersionBellatrix,
				Bellatrix: bellatrixBlock,
			}

			return nil, p.SubmitBeaconBlock(ctx, block)
		}

		altairBlock := new(altair.SignedBeaconBlock)
		err = altairBlock.UnmarshalJSON(body)
		if err == nil {
			block := &eth2spec.VersionedSignedBeaconBlock{
				Version: eth2spec.DataVersionAltair,
				Altair:  altairBlock,
			}

			return nil, p.SubmitBeaconBlock(ctx, block)
		}

		phase0Block := new(eth2p0.SignedBeaconBlock)
		err = phase0Block.UnmarshalJSON(body)
		if err == nil {
			block := &eth2spec.VersionedSignedBeaconBlock{
				Version: eth2spec.DataVersionPhase0,
				Phase0:  phase0Block,
			}

			return nil, p.SubmitBeaconBlock(ctx, block)
		}

		return nil, errors.New("invalid submitted block", z.Hex("body", body))
	}
}

func submitBlindedBlock(p eth2client.BlindedBeaconBlockSubmitter) handlerFunc {
	return func(ctx context.Context, params map[string]string, query url.Values, body []byte) (interface{}, error) {
		// The blinded block maybe either bellatrix or capella.
		capellaBlock := new(eth2capella.SignedBlindedBeaconBlock)
		err := capellaBlock.UnmarshalJSON(body)
		if err == nil {
			block := &eth2api.VersionedSignedBlindedBeaconBlock{
				Version: eth2spec.DataVersionCapella,
				Capella: capellaBlock,
			}

			return nil, p.SubmitBlindedBeaconBlock(ctx, block)
		}

		bellatrixBlock := new(eth2bellatrix.SignedBlindedBeaconBlock)
		err = bellatrixBlock.UnmarshalJSON(body)
		if err == nil {
			block := &eth2api.VersionedSignedBlindedBeaconBlock{
				Version:   eth2spec.DataVersionBellatrix,
				Bellatrix: bellatrixBlock,
			}

			return nil, p.SubmitBlindedBeaconBlock(ctx, block)
		}

		return nil, errors.New("invalid block")
	}
}

// submitValidatorRegistrations returns a handler function for the validator (builder) registration submitter endpoint.
func submitValidatorRegistrations(r eth2client.ValidatorRegistrationsSubmitter) handlerFunc {
	return func(ctx context.Context, _ map[string]string, _ url.Values, body []byte) (interface{}, error) {
		var unversioned []*eth2v1.SignedValidatorRegistration
		if err := json.Unmarshal(body, &unversioned); err != nil {
			return nil, errors.Wrap(err, "unmarshal signed builder registration")
		}

		var versioned []*eth2api.VersionedSignedValidatorRegistration
		for _, registration := range unversioned {
			versioned = append(versioned, &eth2api.VersionedSignedValidatorRegistration{
				Version: eth2spec.BuilderVersionV1,
				V1:      registration,
			})
		}

		return nil, r.SubmitValidatorRegistrations(ctx, versioned)
	}
}

// aggregateBeaconCommitteeSelections receives partial beacon committee selections and returns aggregated selections.
func aggregateBeaconCommitteeSelections(a eth2exp.BeaconCommitteeSelectionAggregator) handlerFunc {
	return func(ctx context.Context, _ map[string]string, _ url.Values, body []byte) (res interface{}, err error) {
		var selections []*eth2exp.BeaconCommitteeSelection
		if err := json.Unmarshal(body, &selections); err != nil {
			return nil, errors.Wrap(err, "unmarshal beacon committee selections")
		}

		resp, err := a.AggregateBeaconCommitteeSelections(ctx, selections)
		if err != nil {
			return nil, err
		}

		return aggregateBeaconCommitteeSelectionsJSON{Data: resp}, nil
	}
}

// aggregateSyncCommitteeSelections receives partial sync committee selections and returns aggregated selections.
func aggregateSyncCommitteeSelections(a eth2exp.SyncCommitteeSelectionAggregator) handlerFunc {
	return func(ctx context.Context, _ map[string]string, _ url.Values, body []byte) (res interface{}, err error) {
		var selections []*eth2exp.SyncCommitteeSelection
		if err := json.Unmarshal(body, &selections); err != nil {
			return nil, errors.Wrap(err, "unmarshal sync committee selections")
		}

		resp, err := a.AggregateSyncCommitteeSelections(ctx, selections)
		if err != nil {
			return nil, err
		}

		return aggregateSyncCommitteeSelectionsJSON{Data: resp}, nil
	}
}

// submitExit returns a handler function for the exit submitter endpoint.
func submitExit(p eth2client.VoluntaryExitSubmitter) handlerFunc {
	return func(ctx context.Context, _ map[string]string, _ url.Values, body []byte) (interface{}, error) {
		exit := new(eth2p0.SignedVoluntaryExit)
		if err := exit.UnmarshalJSON(body); err != nil {
			return nil, errors.Wrap(err, "unmarshal signed voluntary exit")
		}

		return nil, p.SubmitVoluntaryExit(ctx, exit)
	}
}

func tekuProposerConfig(p TekuProposerConfigProvider) handlerFunc {
	return func(ctx context.Context, _ map[string]string, _ url.Values, _ []byte) (interface{}, error) {
		return p.TekuProposerConfig(ctx)
	}
}

func aggregateAttestation(p eth2client.AggregateAttestationProvider) handlerFunc {
	return func(ctx context.Context, params map[string]string, query url.Values, _ []byte) (interface{}, error) {
		slot, err := uintQuery(query, "slot")
		if err != nil {
			return nil, err
		}

		var attDataRoot eth2p0.Root
		if err := hexQueryFixed(query, "attestation_data_root", attDataRoot[:]); err != nil {
			return nil, err
		}

		data, err := p.AggregateAttestation(ctx, eth2p0.Slot(slot), attDataRoot)
		if err != nil {
			return nil, err
		}

		return struct {
			Data *eth2p0.Attestation `json:"data"`
		}{
			Data: data,
		}, nil
	}
}

func submitAggregateAttestations(s eth2client.AggregateAttestationsSubmitter) handlerFunc {
	return func(ctx context.Context, _ map[string]string, _ url.Values, body []byte) (interface{}, error) {
		var aggs []*eth2p0.SignedAggregateAndProof
		err := json.Unmarshal(body, &aggs)
		if err != nil {
			return nil, errors.Wrap(err, "unmarshal signed aggregate and proofs")
		}

		err = s.SubmitAggregateAttestations(ctx, aggs)
		if err != nil {
			return nil, err
		}

		return nil, nil
	}
}

func submitSyncCommitteeMessages(s eth2client.SyncCommitteeMessagesSubmitter) handlerFunc {
	return func(ctx context.Context, params map[string]string, query url.Values, body []byte) (interface{}, error) {
		var msgs []*altair.SyncCommitteeMessage
		err := json.Unmarshal(body, &msgs)
		if err != nil {
			return nil, errors.Wrap(err, "unmarshal sync committee messages")
		}

		err = s.SubmitSyncCommitteeMessages(ctx, msgs)
		if err != nil {
			return nil, err
		}

		return nil, nil
	}
}

// submitProposalPreparations swallows fee-recipient-address from validator client as it should be
// configured by charon from cluster-lock.json and VC need not be configured with correct fee-recipient-address.
func submitProposalPreparations() handlerFunc {
	return func(context.Context, map[string]string, url.Values, []byte) (interface{}, error) {
		return nil, nil
	}
}

// proxyHandler returns a reverse proxy handler.
func proxyHandler(eth2Cl eth2wrap.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get active beacon node address.
		targetURL, err := getBeaconNodeAddress(r.Context(), eth2Cl)
		if err != nil {
			ctx := log.WithTopic(r.Context(), "vapi")
			log.Error(ctx, "Proxy target beacon node address", err)
			writeError(ctx, w, "proxy", err)

			return
		}
		// Get address for active beacon node
		proxy := httputil.NewSingleHostReverseProxy(targetURL)

		// Extend default proxy director with basic auth and host header.
		defaultDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			if targetURL.User != nil {
				password, _ := targetURL.User.Password()
				req.SetBasicAuth(targetURL.User.Username(), password)
			}
			req.Host = targetURL.Host
			defaultDirector(req)
		}
		proxy.ErrorLog = stdlog.New(io.Discard, "", 0)

		defer observeAPILatency("proxy")()
		proxy.ServeHTTP(proxyResponseWriter{w.(writeFlusher)}, r)
	}
}

// getBeaconNodeAddress returns an active beacon node proxy target address.
func getBeaconNodeAddress(ctx context.Context, eth2Cl eth2wrap.Client) (*url.URL, error) {
	addr := eth2Cl.Address()
	if addr == "none" {
		// Trigger refresh of inactive clients to hopefully resolve any active clients.
		syncProvider, ok := eth2Cl.(eth2client.NodeSyncingProvider)
		if !ok {
			return nil, errors.New("invalid eth2 client")
		}
		_, err := syncProvider.NodeSyncing(ctx)
		if err != nil {
			return nil, errors.New("no active beacon nodes") // Not wrapping since error will be confusing.
		}

		addr = eth2Cl.Address()
		if addr == "none" {
			return nil, errors.New("no active beacon nodes")
		}
	}

	targetURL, err := url.Parse(addr)
	if err != nil {
		return nil, errors.Wrap(err, "invalid beacon node address", z.Str("address", addr))
	}

	return targetURL, nil
}

// writeResponse writes the 200 OK response and json response body.
func writeResponse(ctx context.Context, w http.ResponseWriter, endpoint string, response interface{}) {
	w.WriteHeader(http.StatusOK)

	if response == nil {
		return
	}

	b, err := json.Marshal(response)
	if err != nil {
		writeError(ctx, w, endpoint, errors.Wrap(err, "marshal response body"))
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if _, err = w.Write(b); err != nil {
		// Too late to also try to writeError at this point, so just log.
		log.Error(ctx, "Failed writing api response", err)
	}
}

// writeError writes a http json error response object.
func writeError(ctx context.Context, w http.ResponseWriter, endpoint string, err error) {
	if ctx.Err() != nil {
		// Client cancelled the request
		err = apiError{
			StatusCode: http.StatusRequestTimeout,
			Message:    "client cancelled request",
			Err:        ctx.Err(),
		}
	}

	var aerr apiError
	if !errors.As(err, &aerr) {
		aerr = apiError{
			StatusCode: http.StatusInternalServerError,
			Message:    "Internal server error",
			Err:        err,
		}
	}

	if aerr.StatusCode/100 == 4 {
		// 4xx status codes are client errors (not server), so log as debug only.
		log.Debug(ctx, "Validator api 4xx response",
			z.Int("status_code", aerr.StatusCode),
			z.Str("message", aerr.Message),
			z.Err(err),
			getCtxDuration(ctx))
	} else {
		// 5xx status codes (or other weird ranges) are server errors, so log as error.
		log.Error(ctx, "Validator api 5xx response", err,
			z.Int("status_code", aerr.StatusCode),
			z.Str("message", aerr.Message),
			getCtxDuration(ctx))
	}

	incAPIErrors(endpoint, aerr.StatusCode)

	res := errorResponse{
		Code:    aerr.StatusCode,
		Message: aerr.Message,
		// TODO(corver): Add support for debug mode error and stacktraces.
	}

	b, err2 := json.Marshal(res)
	if err2 != nil {
		// Log and continue to write nil b.
		log.Error(ctx, "Failed marshalling error response", err2)
	}

	w.WriteHeader(aerr.StatusCode)
	w.Header().Set("Content-Type", "application/json")

	if _, err2 = w.Write(b); err2 != nil {
		log.Error(ctx, "Failed writing api error", err2)
	}
}

// unmarshal parses the JSON-encoded request body and stores the result
// in the value pointed to by v.
func unmarshal(body []byte, v interface{}) error {
	if len(body) == 0 {
		return apiError{
			StatusCode: http.StatusBadRequest,
			Message:    "empty request body",
			Err:        errors.New("empty request body"),
		}
	}

	err := json.Unmarshal(body, v)
	if err != nil {
		return apiError{
			StatusCode: http.StatusBadRequest,
			Message:    "failed parsing request body",
			Err:        err,
		}
	}

	return nil
}

// uintParam returns a uint path parameter.
func uintParam(params map[string]string, name string) (uint64, error) {
	param, ok := params[name]
	if !ok {
		return 0, apiError{
			StatusCode: http.StatusBadRequest,
			Message:    fmt.Sprintf("missing path parameter %s", name),
		}
	}

	res, err := strconv.ParseUint(param, 10, 64)
	if err != nil {
		return 0, apiError{
			StatusCode: http.StatusBadRequest,
			Message:    fmt.Sprintf("invalid uint path parameter %s [%s]", name, param),
			Err:        err,
		}
	}

	return res, nil
}

// uintQuery returns a uint query parameter.
func uintQuery(query url.Values, name string) (uint64, error) {
	if !query.Has(name) {
		return 0, apiError{
			StatusCode: http.StatusBadRequest,
			Message:    fmt.Sprintf("missing query parameter %s", name),
		}
	}

	param := query.Get(name)

	res, err := strconv.ParseUint(param, 10, 64)
	if err != nil {
		return 0, apiError{
			StatusCode: http.StatusBadRequest,
			Message:    fmt.Sprintf("invalid uint path parameter %s [%s]", name, param),
			Err:        err,
		}
	}

	return res, nil
}

// hexQueryFixed parses a fixed length 0x-hex query parameter into target.
func hexQueryFixed(query url.Values, name string, target []byte) error {
	resp, ok, err := hexQuery(query, name)
	if err != nil {
		return err
	} else if !ok {
		return apiError{
			StatusCode: http.StatusBadRequest,
			Message:    fmt.Sprintf("missing 0x-hex query parameter %s", name),
		}
	} else if len(resp) != len(target) {
		return apiError{
			StatusCode: http.StatusBadRequest,
			Message:    fmt.Sprintf("invalid length for 0x-hex query parameter %s, expect %d bytes", name, len(target)),
		}
	}
	copy(target, resp)

	return nil
}

// hexQuery returns a 0x-prefixed hex query parameter with name or false if not present.
func hexQuery(query url.Values, name string) ([]byte, bool, error) {
	valueA, ok := query[name]
	if !ok || len(valueA) != 1 {
		return nil, false, nil
	}
	value := valueA[0]

	resp, err := hex.DecodeString(strings.TrimPrefix(value, "0x"))
	if err != nil {
		return nil, false, apiError{
			StatusCode: http.StatusBadRequest,
			Message:    fmt.Sprintf("invalid 0x-hex query parameter %s [%s]", name, value),
			Err:        err,
		}
	}

	return resp, true, nil
}

// writeFlusher is copied from /net/http/httputil/reverseproxy.go.
// It is required to flush streaming responses.
type writeFlusher interface {
	http.ResponseWriter
	http.Flusher
}

// proxyResponseWriter wraps the writeFlusher interface and instruments errors.
type proxyResponseWriter struct {
	writeFlusher
}

func (w proxyResponseWriter) WriteHeader(statusCode int) {
	if statusCode/100 == 2 {
		// 2XX isn't an error
		return
	}

	incAPIErrors("proxy", statusCode)
	w.writeFlusher.WriteHeader(statusCode)
}

// stubRoot return a stub dependent root for an epoch.
func stubRoot(epoch uint64) root {
	var r eth2p0.Root
	binary.PutUvarint(r[:], epoch)

	return root(r)
}

// getValidatorIDs returns validator IDs as "id" query parameters (supporting csv values).
func getValidatorIDs(query url.Values) []string {
	var resp []string
	for _, csv := range query["id"] {
		for _, id := range strings.Split(csv, ",") {
			resp = append(resp, strings.TrimSpace(id))
		}
	}

	return resp
}

// getValidatorsByID returns the validators with ids being either pubkeys or validator indexes.
func getValidatorsByID(ctx context.Context, p eth2client.ValidatorsProvider, stateID string, ids ...string) ([]v1Validator, error) {
	if len(ids) == 0 {
		return nil, errors.New("no validator ids provided")
	}

	flatten := func(kvs map[eth2p0.ValidatorIndex]*eth2v1.Validator) []v1Validator {
		var vals []v1Validator
		for _, v := range kvs {
			vals = append(vals, v1Validator(*v))
		}

		return vals
	}

	if strings.HasPrefix(ids[0], "0x") {
		var pubkeys []eth2p0.BLSPubKey
		for _, id := range ids {
			pubkey, err := tblsconv.KeyFromCore(core.PubKey(id))
			if err != nil {
				return nil, errors.Wrap(err, "decode public key hex")
			}
			eth2Pubkey, err := tblsconv.KeyToETH2(pubkey)
			if err != nil {
				return nil, err
			}

			pubkeys = append(pubkeys, eth2Pubkey)
		}

		vals, err := p.ValidatorsByPubKey(ctx, stateID, pubkeys)
		if err != nil {
			return nil, err
		}

		return flatten(vals), nil
	}

	var vIdxs []eth2p0.ValidatorIndex
	for _, id := range ids {
		vIdx, err := strconv.ParseUint(id, 10, 64)
		if err != nil {
			return nil, errors.Wrap(err, "parse validator index")
		}
		vIdxs = append(vIdxs, eth2p0.ValidatorIndex(vIdx))
	}

	vals, err := p.Validators(ctx, stateID, vIdxs)
	if err != nil {
		return nil, err
	}

	return flatten(vals), nil
}

type durationKey struct{}

// withCtxDuration returns a copy of parent in which the current time is associated with the duration key.
func withCtxDuration(ctx context.Context) context.Context {
	return context.WithValue(ctx, durationKey{}, time.Now())
}

// getCtxDuration returns a zap field with the duration withCtxDuration was called on the context.
// Else it returns a noop zap field.
func getCtxDuration(ctx context.Context) z.Field {
	v := ctx.Value(durationKey{})
	if v == nil {
		return z.Skip
	}
	t0, ok := v.(time.Time)
	if !ok {
		return z.Skip
	}

	return z.Str("duration", time.Since(t0).String())
}
