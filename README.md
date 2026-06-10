# Back2Past

Trabalho de Redes — rádio TCP em Go que transmite áudio em tempo real para vários ouvintes simultâneos.

## Descrição

O projeto consiste em uma rádio chamada **Back2Past**, uma maneira de relembrar um costume não tão distante de nós.

O servidor da rádio entra ao ar e permite que os clientes estabeleçam uma conexão TCP com ela. Ao conectar, o cliente recebe pacotes de áudio a partir do ponto em que a transmissão está naquele momento — como sintonizar uma rádio FM no meio de uma música.

Esses pacotes podem ser:

- **Reproduzidos em tempo real** no alto-falante
- **Gravados e concatenados** em um único arquivo `.wav` enquanto o cliente permanece conectado

O sistema oferece **quatro executáveis**:

| Executável | Papel | Interface |
|------------|-------|-----------|
| `cmd/studio` | **Locutor** — mixagem música + microfone + servidor TCP | GUI (Fyne) |
| `cmd/gui` | **Ouvinte** — conectar, ouvir, gravar, pausar | GUI (Fyne) |
| `cmd/server` | Servidor simples (arquivo em loop ou microfone) | Terminal |
| `cmd/client` | Ouvinte com comandos `pause`/`resume` | Terminal |

> **Nota:** `cmd/studio` e `cmd/gui` são as aplicações principais para demonstração. `cmd/server` e `cmd/client` são atalhos de terminal que reutilizam as mesmas bibliotecas internas (`broadcast`, `source`, `listener`) — não dependem um do outro.

---

## Tecnologias utilizadas

### Linguagem

| Tecnologia | Versão | Uso no projeto |
|------------|--------|----------------|
| **Go (Golang)** | 1.22+ (projeto usa 1.24) | Servidor, cliente, protocolo e GUIs |

### Bibliotecas e pacotes

| Biblioteca / Pacote | Tipo | Uso no projeto |
|---------------------|------|----------------|
| **`net`** (stdlib) | TCP sockets | Conexão servidor ↔ cliente |
| **`encoding/json`** (stdlib) | Serialização | Metadados de áudio no handshake |
| **`encoding/binary`** (stdlib) | Binário | Framing dos pacotes e cabeçalho WAV |
| **`sync` / goroutines** (stdlib) | Concorrência | Múltiplos clientes, broadcast, buffers |
| **`github.com/ebitengine/oto/v3`** | Externa | Reprodução PCM em tempo real no ouvinte |
| **`fyne.io/fyne/v2`** | Externa | Interface gráfica (`cmd/studio`, `cmd/gui`) |
| **`github.com/gen2brain/malgo`** | Externa | Captura de microfone (estúdio e modo live) |

### Protocolo e comunicação

| Item | Detalhe |
|------|---------|
| **Transporte** | TCP (porta padrão `9090`) |
| **Handshake** | Frame `0x01` com metadados JSON (codec, sample rate, canais, etc.) |
| **Streaming** | Frames `0x02` com blocos de até **4096 bytes** (~23 ms em PCM 44.1 kHz estéreo) |
| **Troca de fonte** | Frame `0x01` reenviado quando o locutor muda música/microfone |
| **Formato transmitido** | PCM cru 16-bit (cabeçalho WAV removido pelo servidor) |
| **Latência TCP** | `TCP_NODELAY` habilitado em servidor e cliente |

### Ferramentas de desenvolvimento

| Ferramenta | Uso |
|------------|-----|
| **Go modules** (`go.mod` / `go.sum`) | Gerenciamento de dependências |
| **`go run` / `go build`** | Compilar e executar |
| **`go test ./...`** | Testes unitários (parser WAV, mix PCM, playlist) |

---

## Arquitetura do código

O código em `internal/` está organizado por **responsabilidade**, não por “lado servidor/cliente”:

```
internal/
├── protocol/              # Comunicação — frames TCP e AudioMeta (JSON)
├── netx/                  # Comunicação — TCP_NODELAY
│
├── audio/
│   ├── pcm/               # Mixagem, silêncio, ring buffer, tamanho de chunks
│   ├── wav/               # Parser RIFF e montagem de cabeçalho WAV
│   ├── pace/              # Relógio de pacing (tempo real)
│   └── capture/           # Configuração e listagem de microfones
│
├── broadcast/             # Servidor TCP — Radio + interface Source
├── source/                # Fontes simples — arquivo em loop, microfone direto
├── studio/                # Mesa do locutor — mixagem música + mic, playlist
├── listener/              # Ouvinte — recepção, playback (oto), gravação WAV
└── ui/                    # Tema e componentes visuais Fyne (design retrô)
```

### Fluxo de dados

```
[cmd/studio]
    studio (mix música + mic) → broadcast.Radio → TCP :9090
                                      ↓
[cmd/gui]                    listener (frames 0x01/0x02)
                                      ↓
                              oto (alto-falante) + output/*.wav
```

---

## Como executar o sistema

> **Importante:** todos os comandos devem ser executados dentro da pasta `Streaming_Audio-Redes` (onde está o `go.mod`).

### Passo 0 — Pré-requisitos

1. **Go instalado** (versão 1.22 ou superior)

   ```bash
   go version
   ```

2. **Arquivos de áudio** na pasta `assets/` (ex.: `musica.wav`, `musica2.wav`)

3. **Dependências** (primeira execução):

   ```bash
   cd Streaming_Audio-Redes
   go mod download
   ```

4. **Permissão de microfone** (macOS) — necessária para o estúdio e modo `-source live`:
   Ajustes → Privacidade e Segurança → Microfone → permitir Terminal ou IDE.

---

### Demonstração principal — Estúdio + GUI (recomendado)

**Terminal 1 — Estúdio do locutor:**

```bash
go run ./cmd/studio -addr :9090 -channels 2
```

Na janela **Cabine do Locutor**:

1. Selecione uma faixa na **playlist** e clique em **▶ Tocar**
2. Ative **Microfone aberto** para falar por cima da música
3. Ajuste os sliders **Voz** e **Música**
4. Use a **barra de tempo** ou **+10s** para avançar na faixa
5. **Fora do ar** — para música e microfone

**Terminal 2 — Ouvinte (GUI):**

```bash
go run ./cmd/gui
```

Na janela **Back2Past**:

1. Endereço: `localhost:9090`
2. Marque **Reproduzir ao vivo** e/ou **Gravar em arquivo**
3. Clique em **▶ Conectar**
4. Use **Pausar** / **Retomar** / **Desconectar** conforme necessário

A gravação padrão vai para `output/ouvindo.wav`.

---

### Alternativa — Servidor e cliente no terminal

**Terminal 1 — Servidor simples (arquivo em loop):**

```bash
go run ./cmd/server -addr :9090 -source file -audio assets/musica.wav
```

**Modo microfone ao vivo:**

```bash
go run ./cmd/server -addr :9090 -source live -channels 2
go run ./cmd/server -list-devices   # listar microfones
```

**Terminal 2 — Cliente terminal:**

```bash
# Ouvir e gravar
go run ./cmd/client -addr localhost:9090 -play -output output/cliente1.wav

# Apenas ouvir
go run ./cmd/client -addr localhost:9090 -play -output -

# Apenas gravar
go run ./cmd/client -addr localhost:9090 -output output/cliente1.wav
```

Comandos no prompt `> ` do cliente terminal:

| Comando | Ação |
|---------|------|
| `pause` ou `p` | Pausa gravação e reprodução local |
| `resume` ou `r` | Retoma gravação e reprodução |
| `status` | Mostra pacotes recebidos e buffer |
| `stop` ou `s` | Encerra o cliente |

**Terminal 3 (opcional) — Segundo ouvinte:**

```bash
go run ./cmd/client -addr localhost:9090 -play -output output/cliente2.wav
```

No estúdio ou nos logs do servidor: `ouvintes ativos: 2`.

---

### Verificar gravação

```bash
afplay output/ouvindo.wav      # macOS
aplay output/ouvindo.wav       # Linux
```

---

## Roteiro de demonstração para o professor

| # | O que demonstrar | Como verificar |
|---|------------------|----------------|
| 1 | Estúdio no ar | Janela do locutor aberta; log `rádio iniciada` |
| 2 | Handshake de metadados | Log `metadados enviados/recebidos` ao conectar |
| 3 | Transmissão em tempo real | Logs `pacote #N enviado`; áudio contínuo no ouvinte |
| 4 | Mixagem locutor | Música + microfone com volumes independentes |
| 5 | Dois ouvintes simultâneos | Contador de ouvintes no estúdio ou log `ouvintes ativos: 2` |
| 6 | Mid-stream join | Novo cliente conecta no meio da faixa — não recomeça do zero |
| 7 | Pausar/retomar | Botões na GUI ou comandos no cliente terminal |
| 8 | Gravação WAV válida | `output/ouvindo.wav` abre e toca normalmente |
| 9 | Troca de faixa | Locutor muda música; ouvinte recebe novo frame `0x01` |

---

## Problemas comuns

| Erro | Causa | Solução |
|------|-------|---------|
| `go.mod file not found` | Pasta errada | `cd Streaming_Audio-Redes` antes de `go run` |
| `no such file: assets/musica.wav` | Áudio ausente | Coloque `.wav` em `assets/` |
| `connection refused` | Servidor não rodando | Inicie `cmd/studio` ou `cmd/server` primeiro |
| `address already in use` | Porta 9090 ocupada | Use `-addr :9091` no servidor e `localhost:9091` no cliente |
| Faixa não entra na playlist | Formato incompatível | WAV PCM **44100 Hz estéreo 16-bit**; use `-channels 2` no estúdio |
| Sem som com `-play` | Arquivo não é PCM | Use `.wav` PCM; MP3 só grava, não reproduz ao vivo |
| Sem captura de microfone | Permissão negada | macOS: permitir microfone para Terminal/IDE |
| `dispositivo de entrada não encontrado` | Nome errado em `-input` | Rode `-list-devices` e use parte do nome listado |
| Áudio com atraso após pause | Buffer antigo | Resume limpa buffers — comportamento esperado |

---

## Funcionalidades implementadas

### Estúdio do locutor (`cmd/studio`)

- [x] GUI com tema retrô (design system de rádio FM)
- [x] Mixagem em tempo real: música + microfone com volumes independentes
- [x] Playlist de faixas WAV com validação de formato
- [x] Barra de tempo com seek e botão +10s
- [x] Servidor TCP embutido (`broadcast.Radio`) na mesma aplicação
- [x] Atualização de metadados mid-stream ao trocar fonte no ar
- [x] Contador de ouvintes e pacotes enviados

### Ouvinte GUI (`cmd/gui`)

- [x] Conexão, pausar, retomar e desconectar
- [x] Reprodução ao vivo e gravação em WAV
- [x] Indicador visual de áudio (VU meter)
- [x] Status de latência (~ms) e fonte no ar
- [x] Tratamento de metadados mid-stream

### Servidor terminal (`cmd/server`)

- [x] Transmissão de arquivo `.wav` em loop (`-source file`)
- [x] Captura ao vivo do microfone (`-source live`)
- [x] Listagem de dispositivos (`-list-devices`)
- [x] Pacing em tempo real e broadcast multi-cliente

### Ouvinte terminal (`cmd/client`)

- [x] Handshake, gravação concatenada em WAV e reprodução (`-play`)
- [x] Comandos interativos: `pause`, `resume`, `stop`, `status`
- [x] Mid-stream join e troca de fonte via metadados

### Protocolo e rede (`internal/protocol`, `internal/broadcast`)

- [x] Framing TCP: `[4 bytes tamanho][1 byte tipo][payload]`
- [x] Frame `0x01` — metadados JSON; frame `0x02` — áudio PCM
- [x] `writeAll` — garante envio completo de cada frame
- [x] `TCP_NODELAY` — reduz latência no streaming
- [x] Broadcast síncrono: um chunk lido, N clientes atendidos

---

## Referência de flags

### Estúdio (`cmd/studio`)

| Flag | Padrão | Descrição |
|------|--------|-----------|
| `-addr` | `:9090` | Endereço TCP do servidor |
| `-input` | *(vazio)* | Microfone (substring do nome; vazio = padrão) |
| `-sample-rate` | `44100` | Taxa de amostragem (Hz) |
| `-channels` | `2` | Canais (2 = estéreo, igual aos WAV em `assets/`) |

### Servidor terminal (`cmd/server`)

| Flag | Padrão | Descrição |
|------|--------|-----------|
| `-addr` | `:9090` | Endereço e porta TCP |
| `-source` | `file` | `file` (loop) ou `live` (microfone) |
| `-audio` | `assets/musica.wav` | Arquivo quando `-source=file` |
| `-input` | *(vazio)* | Dispositivo de entrada no modo live |
| `-sample-rate` | `44100` | Taxa de amostragem no modo live |
| `-channels` | `1` | Canais no modo live |
| `-list-devices` | `false` | Lista microfones e encerra |

### Cliente terminal (`cmd/client`)

| Flag | Padrão | Descrição |
|------|--------|-----------|
| `-addr` | `localhost:9090` | Endereço do servidor |
| `-output` | `output/recebido.wav` | Arquivo de saída (`-` = não gravar) |
| `-play` | `false` | Reproduzir áudio em tempo real |

---

## Protocolo de comunicação

Cada mensagem TCP:

```
[4 bytes: tamanho big-endian][1 byte: tipo][payload]
```

| Tipo | Valor | Quando |
|------|-------|--------|
| Metadados | `0x01` | Ao conectar e quando a fonte muda |
| Áudio | `0x02` | Continuamente (broadcast) |

Exemplo de metadados:

```json
{
  "codec": "pcm",
  "container": "studio",
  "sample_rate": 44100,
  "channels": 2,
  "bits_per_sample": 16,
  "source": "musica.wav + microfone"
}
```

O servidor transmite **somente PCM** (sem cabeçalho RIFF). O ouvinte monta o cabeçalho WAV localmente a partir dos metadados e concatena os pacotes em um arquivo reproduzível.

---

## Estrutura do projeto

```
Streaming_Audio-Redes/
├── go.mod / go.sum
├── cmd/
│   ├── studio/main.go       # locutor — GUI + mixagem + servidor TCP
│   ├── gui/main.go          # ouvinte — GUI
│   ├── server/main.go       # servidor simples (terminal)
│   └── client/main.go       # ouvinte (terminal)
├── internal/
│   ├── protocol/            # frames e metadados JSON
│   ├── netx/                # TCP_NODELAY
│   ├── audio/
│   │   ├── pcm/             # mix, silêncio, ring buffer
│   │   ├── wav/             # parser e cabeçalho WAV
│   │   ├── pace/            # pacing em tempo real
│   │   └── capture/         # config de microfone
│   ├── broadcast/           # Radio (servidor TCP)
│   ├── source/              # fontes file e live
│   ├── studio/              # mesa do locutor
│   ├── listener/            # ouvinte (rede + áudio + gravação)
│   └── ui/                  # tema e componentes Fyne
├── assets/                  # faixas de áudio (.wav)
└── output/                  # gravações dos ouvintes (runtime)
```

---

## Possíveis melhorias futuras

- [ ] Decoder MP3 ao vivo no ouvinte
- [ ] TLS e autenticação na conexão TCP
- [ ] Cliente web (WebSocket / WebRTC)
- [ ] Reconexão automática após queda de rede
- [ ] Empacotamento Docker para deploy
- [ ] Testes de integração servidor + ouvinte

---

## Testes

```bash
go test ./...
```

Cobre parser WAV (chunks RIFF), mixagem PCM e validação de faixas na playlist do estúdio.
