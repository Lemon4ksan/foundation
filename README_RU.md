<img src="assets/logo.svg" align="right" width="128" height="128" alt="foundation" />

# foundation

[English](README.md) • Русский

[![Go Version](https://img.shields.io/badge/go-1.27%2B-007d9c?logo=go&logoColor=white&style=flat-square)](https://go.dev/)
[![Go Reference](https://img.shields.io/badge/godoc-reference-007d9c?style=flat-square)](https://pkg.go.dev/github.com/lemon4ksan/foundation)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue?style=flat-square)](LICENSE)

`foundation` — альтернатива стандартной библиотеке Go для систем, где поведение стандартной библиотеки по умолчанию слишком медленное или производит слишком много аллокаций в куче. Проект пересматривает компромиссы пакетов `net` и `encoding`, жертвуя максимальной простотой ради SIMD-ускорения, явного управления памятью и нулевых аллокаций в горячих путях.

Используется как базовый фундамент для сетевых, серверных и системных решений автора: [`aoni`](https://github.com/lemon4ksan/aoni), [`mach`](https://github.com/lemon4ksan/mach), [`sein`](https://github.com/lemon4ksan/sein), [`g-man`](https://github.com/lemon4ksan/g-man), [`dawn`](https://github.com/lemon4ksan/dawn) и [`seal`](https://github.com/lemon4ksan/seal).

## Архитектурные компромиссы

Подробное обоснование представлено в [docs/VISION.md](docs/VISION.md) и [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

| Компромисс стандартной библиотеки | Что делает foundation | Цена решения |
| :--- | :--- | :--- |
| Замороженный навсегда API | Изменения фиксируются в CHANGELOG. | Версии необходимо фиксировать осознанно. |
| Одинаковый код на любом процессоре | Аппаратный SIMD (AVX2/NEON), ассемблер `c2plan9`. | Основной упор на amd64 и arm64. |
| Универсальная сигнатура для всех | Сигнатуры `Append*` и передача `dst []byte`. | Больший объём API, выше порог входа. |
| Аллокации допустимы | Контракты 0 allocs, off-heap память, `borrow`. | Ручной контроль времени жизни буферов. |

## Правила разработки

Разработка в `foundation` подчиняется строгим Architecture Decision Records ([ADR](docs/adr/README.md)):

1. **Слои ([ADR-0002](docs/adr/0002-layering-and-isolation.md))**: Кодовая база разделена на строгие слои L0-L5. Импорты направлены только сверху вниз. Пакет `silicon/` полностью изолирован. `generic` предназначен только для внешних потребителей.
2. **Бюджеты аллокаций ([ADR-0003](docs/adr/0003-allocation-budgets.md))**: Каждый пакет декларирует уровень аллокаций (`zero`, `amortized`, `bounded`, `unconstrained`), что проверяется через `alloc_test.go`.
3. **Никаких протоколов L7 ([ADR-0006](docs/adr/0006-l7-boundary-and-mach.md))**: Прикладные протоколы (HTTP, HPACK, QPACK, gRPC) вынесены в `mach` и проекты-потребители.
4. **Документация ([ADR-0008](docs/adr/0008-doc-go-canonical-truth.md))**: Каноническая документация пакетов находится исключительно в `doc.go` и `example_test.go`.

## Слои системы

### Слой 0: Аппаратный субстрат (`silicon/`)
Автономные пакеты с нулевыми внутренними зависимостями:
* `simd`, `cpukit`: Векторная обработка AVX2/BMI2/GFNI для сканирования фреймов, битовых операций и арифметики полей Галуа.
* `hexkit`, `bytesconv`, `endian`: SIMD-кодек Hex (13 ГБ/с), zero-copy конвертация строк/чисел и примитивы порядка байт без ветвлений.
* `pool`, `ringbuf`, `offheap`: Многоуровневые арены памяти, lock-free SPSC/MPSC кольцевые буферы, внекучевые страничные буферы и slab-аллокаторы.
* `clock`, `randkit`, `sysnet`, `iouring`, `comkit`: Часы RDTSC/Coarse без системных вызовов, lock-free PRNG, низкоуровневые сокетные/векторные операции, кольца Linux `io_uring` и примитивы таблиц виртуальных методов Windows COM/WinRT.

### Слой 1: Память и структуры
* `borrow`: Поколенческое аффинное владение и borrow-чекеры (`Box`, `Cell`, `Ref`, `Mut`).
* `structures`: Кэш-локальные типизированные структуры данных (`minheap`, `deque`, `ringbuffer`, `bitset`, `disjointset`, `linkedlist`, `trie`).
* `bufkit`, `refkit`: Кольцевые/цепочечные буферы с выравниванием по кэш-линии (64 байта) и безрефлексийные аксессоры/парсеры тегов структур.

### Слой 2: Конкурентность
* `sync`: Адаптивные лимитеры Vegas, пер-ключевые мьютексы (`keylock`), circuit breakers, семафоры, backoff, ленивые значения и TTAS-спинлоки.
* `async`: Контексты `ctxkit` без аллокаций, топологический `lifecycle`, `fsm`, `dedup`, `pipeline`, `pool`, `rate`, `scheduler`, `task` и `logkit`.

### Слой 3: Кодеки и криптография
* `encoding`: Потоковый SIMD `json`, zero-alloc wire-сканер и энкодер `proto`, QUIC `varint`, AVX2 `base64` (31 ГБ/с) и бинарная сериализация `bin`.
* `crypto`: Аппаратно-ускоренные шифры AEAD (`aead`), KDF (`kdf`) и цифровые подписи Ed25519/HMAC (`sign`).
* `text`: Конвертация регистра идентификаторов без аллокаций (`casing`) и потоковые трансформации (`transform`).
* `iokit`: Векторизованный, перематываемый и zero-copy ввод-вывод.

### Слой 4: Сеть и транспорт (`net/`)
* `urlkit`: Высокопроизводительный парсер представлений URL на SIMD, экранирование path/query по RFC 3986 и санитарная обработка хоста (`CleanHost`).
* `dns/wire`, `dns/svcb`: L4 DNS wire кодеки и обнаружение сервисов RFC 9460 SVCB/HTTPS (включая извлечение ECH).
* `tls/cert`: Горячая перезагрузка сертификатов X.509 без блокировок и SPKI-пиннинг.
* `ip`: Ротация IP-адресов без блокировок (`SourceRotator`), классификация подсетей и генерация адресов IPv6 `/64`.

### Слой 5: Утилиты и типы
* `types`: Высокоэнтропийный `uuid` (v4/v7), безопасный `secret` и мягкое приведение типов `values`.
* `timekit`: Быстрые метки времени, секундомеры и форматирование длительностей без аллокаций.
* `fskit`, `pathkit`: Многопоточный обходчик ФС, `mmap`-файлы (`MmapFile`), атомарная запись и zero-alloc работа с путями.
* `testing`: Хелперы ассертов и моков без внешних зависимостей (`assert`, `require`, `mock`, `gomock`).

### Внешний уровень: Хелперы для потребителей
* `generic`: Монадические типы (`Optional`, `Result`), шардированные карты, потокобезопасный `LRU` и TTL `Cache`, `SingleFlight`, ленивый `Stream` (вне графа слоев, для внешних потребителей).

## SIMD-ядра (`c2plan9`)

Накладные расходы CGO исключены благодаря компиляции ядер C/LLVM напрямую в ассемблер Go Plan 9 через `c2plan9`.

| Ядро | Описание | Задержка | Пропускная способность | Аллокации |
| :--- | :--- | :--- | :--- | :--- |
| `IndexCRLFCRLFVector` | Сканирование границы HTTP-заголовков (`\r\n\r\n`) | `4.82 ns/op` | 212 ГБ/с | 0 |
| `ValidUTF8_SWAR` | 64-битный SWAR UTF-8 валидатор | `5.78 ns/op` | 12.6 ГБ/с | 0 |
| `Hex.Encode1KB` | AVX2 кодирование Hex | `78.58 ns/op` | 13.0 ГБ/с | 0 |
| `json.UnmarshalNoCopy` | SIMD JSON декодер | `916.4 ns/op` | - | 12 |

## 📄 Лицензия

Лицензировано под **BSD 3-Clause License**. Подробности в файле [LICENSE](LICENSE).
