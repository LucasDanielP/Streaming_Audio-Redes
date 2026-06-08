# Streaming_Audio-Redes

Trabalho de Redes — rádio TCP em Go que transmite um arquivo de áudio em loop para vários clientes simultâneos.

## Funcionalidades (MVP)

| Requisito | Implementação |
|-----------|----------------|
| Servidor TCP em Go | `cmd/server` |
| Leitura contínua de arquivo de áudio | Loop em `internal/server` |
| Envio em blocos para todos os clientes | Broadcast com frames de 4 KB |
| Múltiplos clientes simultâneos | Mapa thread-safe de conexões |
| Cliente Go (salvar fluxo) | `cmd/client` grava em WAV/MP3 |
| Comandos pause / resume / stop | Prompt interativo no cliente |
| Logs de conexão, desconexão e pacotes | `log` no servidor e no cliente |

## Estrutura

```
cmd/
  server/     # servidor de rádio
  client/     # ouvinte
  gensample/  # gera WAV de teste
internal/
  protocol/   # framing [tamanho][dados]
  server/     # aceita TCP, broadcast
  client/     # recebe fluxo e comandos locais
assets/
  sample.wav       # áudio curto (3 s, gerado)
  sample_long.wav  # áudio longo (15 s, gerado) — padrão do servidor
```

## Pré-requisitos

- Go 1.22+

## Uso rápido

```bash
cd Streaming_Audio-Redes

# 1) Gerar áudios de teste (ou coloque seu .wav/.mp3 em assets/)
go run ./cmd/gensample
go run ./cmd/gensample -out assets/sample_long.wav -duration 15

# 2) Iniciar o servidor (usa sample_long.wav por padrão)
go run ./cmd/server -addr :9090

# 3) Em outro terminal — primeiro cliente
go run ./cmd/client -addr localhost:9090 -output output/cliente1.wav

# 4) Em outro terminal — segundo cliente (mesmo fluxo)
go run ./cmd/client -addr localhost:9090 -output output/cliente2.wav
```

### Reprodução em tempo real

```bash
# Ouvir enquanto recebe (sem gravar)
go run ./cmd/client -addr localhost:9090 -play -output -

# Ouvir e gravar ao mesmo tempo
go run ./cmd/client -addr localhost:9090 -play -output output/cliente1.wav
```

O player usa um **buffer que cresce** conforme pacotes chegam — o som não para quando o trecho inicial acaba; novos pacotes entram no buffer e continuam sendo reproduzidos.

### Comandos no cliente

No prompt `> `:

- `pause` ou `p` — pausa gravação e reprodução local (o fluxo continua chegando)
- `resume` ou `r` — retoma gravação e reprodução
- `stop` ou `s` — encerra o cliente
- `status` — mostra pacotes, bytes gravados e buffer de reprodução

### Flags do cliente

| Flag | Padrão | Descrição |
|------|--------|-----------|
| `-addr` | `localhost:9090` | Endereço do servidor |
| `-output` | `output/recebido.wav` | Arquivo de saída (`-` para não gravar) |
| `-play` | `false` | Reproduzir áudio em tempo real (PCM 16-bit) |

### Flags do servidor

| Flag | Padrão | Descrição |
|------|--------|-----------|
| `-addr` | `:9090` | Porta TCP |
| `-audio` | `assets/sample_long.wav` | Arquivo transmitido em loop |
| `-chunk-ms` | `20` | Intervalo entre blocos (ms) |

### Áudio próprio

Use qualquer `.wav` ou `.mp3`:

```bash
go run ./cmd/server -audio /caminho/para/musica.mp3
```

> A primeira versão reenvia o arquivo do início ao fim em loop, conforme o enunciado.

## Demonstração do MVP

1. Servidor iniciado com logs `rádio iniciada`
2. Dois clientes conectados — logs `ouvintes ativos: 2`
3. Pacotes sendo enviados/recebidos (`pacote #N`)
4. Comandos `pause` / `resume` / `stop` no cliente
5. Arquivos `output/cliente1.wav` reproduzíveis (ex.: `afplay output/cliente1.wav` no macOS)

### Gravação no cliente

O servidor transmite **somente PCM** (pula o cabeçalho `RIFF` do arquivo). O cliente monta o cabeçalho WAV a partir dos metadados e concatena cada pacote de áudio recebido. Ex.: 2 loops de 15 s → arquivo de ~30 s reproduzível.

## Protocolo (Fase 1 + handshake)

Cada mensagem TCP:

```
[4 bytes: tamanho big-endian][1 byte: tipo][payload]
```

| Tipo | Valor | Quando |
|------|-------|--------|
| Metadados | `0x01` | Uma vez, ao conectar (só para aquele cliente) |
| Áudio | `0x02` | Continuamente (broadcast para todos) |

Exemplo de metadados (JSON):

```json
{"codec":"pcm","container":"wav","sample_rate":44100,"channels":1,"bits_per_sample":16,"source":"sample_long.wav"}
```

O cliente usa os metadados para montar o cabeçalho WAV localmente e gravar desde o 1º pacote de áudio, mesmo entrando no meio do loop.

## Próximos passos (fora do MVP)

- Captura de áudio ao vivo (microfone)
- Sincronização mais fina com RTP
- Reprodução em tempo real no cliente (sem só gravar)
