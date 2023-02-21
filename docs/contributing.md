# Charon'a katkı

🎉 Zaman ayırıp katkıda bulunduğunuz için teşekkürler, gerçekten müteşekkiriz.

Başlamak için [Obol Belgelerine](https://docs.obol.tech/) ve diğer Charon deposuna [belgelere](.) bakın.

Katkı sürecini düzene sokmak için basit bir kurallar dizisini tutuyoruz.
genel olarak [Atom katkıda bulunma kılavuzuna](https://github.com/atom/atom/blob/master/CONTRIBUTING.md) dayalıdır.

## Sorumlu Açıklama

⚠️ Kullanıcılarımızın güvenliğini son derece ciddiye alıyoruz.
Bir güvenlik sorunu bulduğunuzu düşünüyorsanız, lütfen bunu instead of opening a public issue or PR on GitHub **sorumlu bir şekilde** `security@obol.tech` adresine bildirin.


## İş akışlarının koordinasyonu

- Bir hata bulduysanız...
     - GitHub'da aynı sorunla ilgili mevcut hata raporlarını kontrol edin.
     - Obol kullanıcılarını korumak için şüphe edilen bir güvenlik açığıysa, bu konuda herkese açık bir şekilde gönderi paylaşmayın;
       bunun yerine "security@obol.tech" kullanın.
     - Teknik bir sorun görüp görmediğinizden emin değilseniz ilgili topluluk kanallarına bir mesaj gönderebilirsiniz.
     - Her şey yolundaysa bir GitHub sorunu açın 🤓
- Mantıklı olan küçük bir değişiklik mi düşünüyorsun? PR göndermekten çekinmeyin.
- Daha büyük bir özellik planlıyorsanız veya yalnızca bir tartışma arıyorsanız,
   [Obol Discord](https://discord.gg/n6ebKsX46w/) içerisinde `#dev-community` altında sohbet edelim.
     - Kodlamadan önce hızlı bir senkronizasyon, çakışan işleri önler ve büyük PR'lerin kabul edilme olasılığını çok daha artırır.
     - 👀 Spam önlemek için Discord kanalı şu anda _davetli_ olarak açılmıştır. Erişim elde etmek için lütfen bir ekip üyesine ping atın.

## Değişiklikleri gönderme

### Topluluk Katkıları (Pull İstekleri)

Charon deposunu fork yapmaktan çekinmeyin ve önerilen değişikliklerinizle birlikte bir pull isteği gönderin.

Ardından, bir yorum olarak görünecek olan Katılımcı Lisans Sözleşmemizi (CLA) imzalamanız gerekecektir.
Açtıktan sonra bu pull isteğinde bir bottan İmzalı bir CLA olmadan kodu inceleyemeyiz.

Yukarıda belirtildiği gibi, bu çekme isteği önemsiz değilse ve ekibimizin anlaması için bağlam gerektiriyorsa lütfen ilgili bir sorun bildirin. Tüm özellikler ve çoğu hata düzeltmesi, tartışılan ve üzerinde karar verilen bir tasarımla ilgili bir soruna sahip olmalıdır. Küçük hata düzeltmeleri ve dokümantasyon iyileştirmeleri sorun gerektirmez.

Yeni özellikler ve hata düzeltmeleri mutlaka test edilmelidir. Belgelerin güncellenmesi gerekebilir. Neyi güncelleyeceğinizden emin değilseniz, PR'yi açın, inceleme sırasında tartışacağız.

Bağımlılıkları ve yeni Go sürümlerini güncelleyen PR'lerin kabul edilmediğini unutmayın. Lütfen bunun yerine bir sorun bildirin.

Not: PR'ler yalnızca obol-buldozer botu tarafından birleştirilebilir. Onay alındıktan sonra `hazır olduğunda birleştir' etiketinin eklenmesi yazarın sorumluluğundadır.

> TL;DR: Ayrıntıları ve bir PR'ın arkasındaki motivasyonu içeren bir Sayı açın.

### Çekirdek Geliştirme Katkıları

- Çalışmanızı bu [charon repo](https://github.com/ObolNetwork/charon) altında bir şubede yayınlayın.
- GitHub [sorunlar](https://github.com/ObolNetwork/charon/issues) aracılığıyla tüm yol haritası ve özellik çalışmalarının yanı sıra hata düzeltmelerini ve daha küçük değişiklikleri takip edin.
- Önerilen dal adları: `<ad>/<özellik>`, ör. "oisin/improve-docs" veya "richard/fix-discv5-panic".
- Git'i "obol.tech" e-postanızı kullanacak şekilde yapılandırın.

## Stil Kılavuzu

### Kararlı gövde üzerinde mikro taahhütler

> TL;DR: Değişimi küçük artışlarla tanıtın

- Dallanma ve yayınları nasıl yaptığımız hakkında daha fazla ayrıntı için lütfen [Dallanma ve Sürüm Modeli](branching.md) belgesine bakın.
- Sayı başına birden çok PR teşvik edilir.
- Bu, gözden geçirilmesi, birleştirilmesi ve test edilmesi kolay küçük PR'ler sağlar.
- Tüm PR'ler tamamlandıktan sonra konu kapatılabilir.
- Sayıdaki bir kontrol listesi aracılığıyla tamamlanmış ve planlanan PR'ları takip etmek harika bir fikir.

### PR Şablonu

- **PR'ler her zaman squash ile main'e birleştirilir**.
- PR başlığı ve gövdesi, nihai squash birleştirilmiş git taahhüt mesajı olarak kullanılır.
- PR'nin orijinal git taahhütleri bu nedenle kaybolur (bu nedenle adlandırma belirtilmez)
- **PR başlık formatı** şu şekilde tanımlanır:
   - [go team's commit format](https://github.com/golang/go/commits/master): "paket[/path]: değişikliğin kısa özeti"
   - Önek, değişiklikten etkilenen birincil paketi tanımlar.
   - Önek, tek veya çift hiyerarşik bir paket adı olabilir, ancak üç veya daha fazla olamaz. Örneğin. "uygulama" veya "uygulama/izleyici".
   - Başlığın geri kalanı, şimdiki zamanda ve küçük harfle başlayan özlü, üst düzey bir genel bakış olmalıdır.
- **PR gövde formatı** şu şekilde tanımlanır:
   - Değişikliğin ayrıntılı açıklamasıyla başlayın.
   - Açıklama, şimdiki zamanda uygun dilbilgisi kullanmalıdır.
   - Bir etiket listesiyle biter (bazıları gerekli, diğerleri isteğe bağlıdır) (`^tag: bu etiketin değeri\n`):
   - "kategori": gerekli; şunlardan biri: "refactor", "bug", "feature", "docs", "release", "tidy", "fixbuild".
   - `bilet`: gerekli; Github sorununun URL'si yalnızca bir referanstır, Örn. "#123" veya "yok".
   - `feature_flag`: isteğe bağlı; özellik bayrağı ("app/featureset" paketine göre) bu kodu etkinleştirir/devre dışı bırakır.
- Örnekler:
```
koşucu/izleyici: jaeger otel ihracatçısını ekleyin

Jaeger ihracatçısını açık telemetre altyapımıza ekler.
category: feature
ticket: #206
feature_flag: jaeger_tracing
```
```
docs: improve contributing.md

`contributing.md`deki yazım hatalarını düzeltin ve dili iyileştirin.
category: docs
ticket: none
```

### Geliştirme araçları, git kancaları ve filtreler.

Charon, çekmeyi sağlayan [pre-commit](https://pre-commit.com) **githooks** ile yapılandırılmıştır
istekler minimum bir standarda bağlıdır ve tutarlıdır. Githooks'lar bir GitHub eylemi olarak çalıştırılır
her PR taahhüdü için. Ancak daha hızlı geri bildirim için geliştirme sırasında githook'ların yerel olarak çalıştırılması önemle tavsiye edilir.

Githook'ları kurmak için:
- "pre-commit" aracını kurmak için [buradaki](https://pre-commit.com/#installation) kurulum talimatlarını izleyin.
- Kurulduktan sonra, projenin kök dizininde `pre-commit install` komutunu çalıştırın. Bu, kancaları kuracaktır.
- `-n` ile taahhütte bulunarak kancaları atlayabileceğinizi unutmayın: `git commit -n -m "look mom no githooks"`

Githook'ları güncellemek için:
```sh
ön taahhüt temizliği
```

Kullanılan **linter** [golangci-lint](https://golangci-lint.run/) şeklindedir. Githook'ların bir parçası olarak çalışır ve [.golangci.yml](../.golangci.yml) içinde yapılandırılır.

Kod tabanında farklı **geliştirme araçları** kullanılır ve bunlar [tools.go](../tools.go) adresinden tanımlanır ve kurulur. Geliştirme araçlarını yüklemek için şunu çalıştırın: `go createtools.go`

## Kod incelemesi
Aşağıdaki kod inceleme yapısını yakından takip etme eğilimindeyiz:
<div><img src="./images/code_review_pyramid.svg" /></div>

## Herhangi bir şey eksik mi?

Bu canlı bir belge. Katkı kılavuzunu geliştirmekte özgürsünüz.
