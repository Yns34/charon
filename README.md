<div align="center"><img src="./docs/images/charonlogo.svg" /></div>
<h1 align="center">Charon<br/>The Distributed Validator middleware client</h1>

<p align="center"><a href="https://github.com/obolnetwork/charon/releases/"><img src="https://img.shields.io/github/tag/obolnetwork/charon.svg"></a>
<a href="https://github.com/ObolNetwork/charon/blob/main/LICENSE"><img src="https://img.shields.io/github/license/obolnetwork/charon.svg"></a>
<a href="https://godoc.org/github.com/obolnetwork/charon"><img src="https://godoc.org/github.com/obolnetwork/charon?status.svg"></a>
<a href="https://goreportcard.com/report/github.com/obolnetwork/charon"><img src="https://goreportcard.com/badge/github.com/obolnetwork/charon"></a>
<a href="https://github.com/ObolNetwork/charon/actions/workflows/golangci-lint.yml"><img src="https://github.com/obolnetwork/charon/workflows/golangci-lint/badge.svg"></a></p>

Bu depo, dağıtılmış doğrulama istemcisi _Charon_ ('kharon' olarak telaffuz edilir) için kaynak kodunu içerir; Ethereum Staking için, bir grup bağımsız düğümde tek bir doğrulayıcıyı güvenli bir şekilde çalıştırmanıza olanak tanıyan bir HTTP ara yazılım istemcisi.

Charon'a dağıtılmış doğrulayıcı anahtar oluşturma için [Dağıtılmış Doğrulayıcı Başlatma Çubuğu](https://goerli.launchpad.obol.tech/) adlı bir web uygulaması eşlik ediyor.
Charon, staker'lar tarafından Ethereum Doğrulayıcılarını çalıştırma sorumluluğunu bir dizi farklı örnek ve istemci uygulamasında dağıtmak için kullanılır.

![Example Obol Cluster](./docs/images/DVCluster.png)

###### İstemci ve donanım hatası risklerinden korunmak için Charon istemcisini kullanan Dağıtılmış Doğrulayıcı Kümesi

## Hızlı başlangıç

Charon'u test etmenin en kolay yolu [charon-distributed-validator-cluster](https://github.com/ObolNetwork/charon-distributed-validator-cluster) deposudur. Yerel makinenizde tam bir charon kümesi çalıştırmak için bir docker oluşturma kurulumu içerir.

## Dokümantasyon

[Obol Belgeleri](https://docs.obol.tech/) web sitesi, başlamak için en iyi yerdir.
Önemli bölümler [intro](https://docs.obol.tech/docs/intro),
[anahtar kavramlar](https://docs.obol.tech/docs/int/key-concepts) ve [charon](https://docs.obol.tech/docs/dv/introduction-charon).
For detailed documentation on this repo, see the [docs](docs) folder:

- [Yapılandırma](docs/configuration.md): Bir charon düğümünün yapılandırılması
- [Architecture](docs/architecture.md): Charon kümesi ve düğüm mimarisine genel bakış
- [Proje Yapısı](docs/structure.md): Proje klasörü yapısı
- [Dallanma ve Yayınlama Modeli](docs/branching.md): Git dallanma ve yayınlanma modeli
- [Go Yönergeleri](docs/goguidelines.md): go geliştirmeye ilişkin yönergeler ve ilkeler
- [Katkıda Bulunma](docs/contributing.md): Charon'a nasıl katkıda bulunulur; githook'lar, PR şablonları vb.

Kaynak kodu dokümantasyonu için her zaman [charon godocs](https://pkg.go.dev/github.com/obolnetwork/charon) vardır.

## Desteklenen Consensus Layer İstemcileri

Charon, doğrulama istemcisi arasında bir ara katman yazılımı olarak Ethereum konsensüs yığınına entegre olur.
ve resmi [Eth Beacon Node REST API](https://ethereum.github.io/beacon-APIs/#/) yoluyla işaret düğümü.
Charon, Beacon API'sine hizmet eden herhangi bir yukarı akış işaret düğümünü destekler.
Charon, Beacon API'sini kullanan herhangi bir aşağı akış bağımsız doğrulayıcı istemcisini desteklemeyi amaçlamaktadır.

| Client                                             | Beacon Node | Validator Client | Notes                                   |
| -------------------------------------------------- | :---------: | :--------------: |-----------------------------------------|
| [Teku](https://github.com/ConsenSys/teku)          |     ✅      |        ✅        | Tam destekli                        |
| [Lighthouse](https://github.com/sigp/lighthouse)   |     ✅      |        ✅        | Tam destekli                       |
| [Lodestar](https://github.com/ChainSafe/lodestar)  |     ✅      |       \*️⃣        | DVT uyumluluk sorunu              |
| [Vouch](https://github.com/attestantio/vouch)      |     \*️⃣     |        ✅        | Sağlanan yalnızca doğrulayıcı istemci          |
| [Prysm](https://github.com/prysmaticlabs/prysm)    |     ✅      |        🛑        | Validator istemcisi, gRPC API gerektirir      |
| [Nimbus](https://github.com/status-im/nimbus-eth2) |     ✅      |        ✅        | Yakında desteklenecek |

## Proje durumu

Obol Ağı için henüz ilk günler ve işler aktif olarak geliştiriliyor.
Hızlı ilerliyoruz, bu nedenle ilerlemeyi takip etmek için düzenli olarak kontrol edin.

Charon dağıtılmış bir doğrulayıcıdır, dolayısıyla ana sorumluluğu doğrulama görevlerini yerine getirmektir.
Aşağıdaki tablo, hangi müşterilerin hangi görevleri genel bir test ağında ürettiğini ve hangilerinin hala yapım aşamasında olduğunu özetlemektedir.(🚧 )

| Duty \ Client                        |                      Teku                      |                    Lighthouse                    | Lodestar | Nimbus | Vouch | Prysm |
|--------------------------------------|:----------------------------------------------:|:------------------------------------------------:|:--------:|:------:|:-----:|:-----:|
| _Attestation_                        |                       ✅                        |                        ✅                         |    🚧    |   🚧   |  ✅   |  🚧   |
| _Attestation Aggregation_            |                       🚧                       |                        🚧                        |    🚧    |   🚧   |  🚧   |  🚧   |
| _Block Proposal_                     |                       ✅                        |                        ✅                         |    🚧    |   🚧   |  🚧   |  🚧   |
| _Blinded Block Proposal (mev-boost)_ | [✅](https://ropsten.beaconcha.in/block/555067) | [✅](https://ropsten.etherscan.io/block/12822070) |    🚧    |   🚧   |  🚧   |  🚧   |
| _Sync Committee Message_             |                       ✅                        |                        ✅                         |    🚧    |   🚧   |  🚧   |  🚧   |
| _Sync Committee Contribution_        |                       🚧                       |                        🚧                        |    🚧    |   🚧   |  🚧   |  🚧   |
