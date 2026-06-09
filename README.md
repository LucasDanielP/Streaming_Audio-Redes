# Back2Past

Trabalho de Redes — rádio TCP em Go que transmite um arquivo de áudio em loop para vários clientes simultâneos.

## Descrição

O projeto consiste em uma rádio chamada **Back2Past**, uma maneira de relembrar um costume não tão distante de nós.

O servidor da rádio entra ao ar e permite que os clientes estabeleçam uma conexão TCP com ela. Ao conectar, o cliente recebe pacotes de áudio a partir do ponto em que a transmissão está naquele momento — como sintonizar uma rádio FM no meio de uma música.

Esses pacotes podem ser:

- **Reproduzidos em tempo real** no alto-falante (`-play`)
- **Gravados e concatenados** em um único arquivo `.wav` enquanto o cliente permanece conectado

O servidor pode transmitir de duas formas:

- **Arquivo em loop** — `.wav` ou `.mp3` fixo (modo padrão)
- **Captura ao vivo** — microfone do sistema transmitido em tempo real via TCP (`-source live`)

---

## Tecnologias utilizadas

### Linguagem

| Tecnologia | Versão | Uso no projeto |
|------------|--------|----------------|
| **Go (Golang)** | 1.22+ (projeto usa 1.24) | Servidor, cliente e protocolo de comunicação |

### Bibliotecas e pacotes

| Biblioteca / Pacote | Tipo | Uso no projeto |
|---------------------|------|----------------|
| **`net`** (stdlib) | TCP sockets | Conexão servidor ↔ cliente |
| **`encoding/json`** (stdlib) | Serialização | Metadados de áudio no handshake |
| **`encoding/binary`** (stdlib) | Binário | Framing dos pacotes e cabeçalho WAV |
| **`sync` / goroutines** (stdlib) | Concorrência | Múltiplos clientes, broadcast, buffer de áudio |
| **`github.com/ebitengine/oto/v3`** | Externa | Reprodução PCM em tempo real no alto-falante |
| **`fyne.io/fyne/v2`** | Externa | Interface gráfica desktop (`cmd/gui`) |
| **`github.com/gen2brain/malgo`** | Externa | Captura de áudio ao vivo no servidor (microfone) |
| **`github.com/ebitengine/purego`** | Externa (dependência do oto) | Acesso ao driver de áudio do sistema |
| **`golang.org/x/sys`** | Externa (dependência do oto) | Chamadas de sistema para áudio |

### Protocolo e comunicação

| Item | Detalhe |
|------|---------|
| **Transporte** | TCP (porta padrão `9090`) |
| **Handshake** | Frame `0x01` com metadados JSON (codec, sample rate, canais, etc.) |
| **Streaming** | Frames `0x02` com blocos de até 1 KB de áudio em tempo real |
| **Troca de fonte** | Frame `0x01` reenviado quando o locutor muda microfone/música |
| **Formato transmitido** | PCM cru (cabeçalho WAV do arquivo é removido pelo servidor) |

### Ferramentas de desenvolvimento

| Ferramenta | Uso |
|------------|-----|
| **Go modules** (`go.mod` / `go.sum`) | Gerenciamento de dependências |
| **`go run` / `go build`** | Compilar e executar servidor e cliente |

---

## Como executar o sistema

> **Importante:** todos os comandos abaixo devem ser executados dentro da pasta `Streaming_Audio-Redes` (onde está o arquivo `go.mod`).

### Passo 0 — Pré-requisitos

1. **Go instalado** (versão 1.22 ou superior)

   ```bash
   go version
   ```

   Saída esperada: `go version go1.22...` ou superior.

2. **Arquivo de áudio** na pasta `assets/`

   O repositório deve conter pelo menos um arquivo `.wav` ou `.mp3`, por exemplo:

   ```
   assets/musica.wav
   ```

   Se o arquivo tiver outro nome, use o parâmetro `-audio` ao iniciar o servidor (veja Passo 2).

3. **Clonar ou baixar o projeto** e entrar na pasta correta:

   ```bash
   cd Streaming_Audio-Redes
   ```

4. **Baixar dependências** (primeira execução ou após clonar):

   ```bash
   go mod download
   ```

---

### Passo 1 — Iniciar o servidor (Terminal 1)

**Modo estúdio do locutor (recomendado — microfone + playlist):**

```bash
cd Streaming_Audio-Redes
go run ./cmd/studio -addr :9090
```

Na janela **Estúdio do Locutor**:

1. **Tocar selecionada** — inicia uma faixa da playlist
2. **Microfone no ar** — ativa/desativa a voz **por cima da música**
3. **Sliders Voz / Música** — controlam o volume de cada fonte no mix
4. **Adiantar +10s** — pula a faixa em 10 segundos
5. **Parar música** / **Silêncio total** — para faixa ou tudo
6. **Adicionar música…** — WAV PCM **44100 Hz estéreo 16-bit** (como `assets/musica.wav`)

O envio é em **tempo real** (cada pacote espera o tempo equivalente de áudio). O cliente limita o buffer a ~500 ms para ficar próximo do ao vivo.

---

**Modo arquivo (loop de `.wav` / `.mp3`):**

```bash
cd Streaming_Audio-Redes
go run ./cmd/server -addr :9090 -source file -audio assets/musica.wav
```

**Logs esperados:**

```
[servidor] rádio iniciada em :9090 | fonte: musica.wav | formato: pcm 44100Hz 1ch 16bit
```

**Modo rádio ao vivo (microfone):**

```bash
go run ./cmd/server -addr :9090 -source live
```

Liste os microfones disponíveis:

```bash
go run ./cmd/server -list-devices
```

Exemplo com dispositivo e taxa de amostragem:

```bash
go run ./cmd/server -addr :9090 -source live -input "MacBook" -sample-rate 44100 -channels 1
```

> Deixe este terminal **aberto e rodando**. No modo arquivo, o servidor transmite em loop; no modo live, captura o microfone continuamente.

**Se o arquivo de áudio tiver outro nome:**

```bash
go run ./cmd/server -addr :9090 -source file -audio assets/musica2.wav
```

---

### Passo 2 — Cliente com interface gráfica (alternativa)

Requer dependências da GUI instaladas (`go mod download`).

```bash
cd Streaming_Audio-Redes

# Terminal 1 — servidor (obrigatório)
go run ./cmd/server -addr :9090 -audio assets/musica.wav

# Terminal 2 — app desktop
go run ./cmd/gui
```

Na janela **Back2Past**: informe o endereço, marque reproduzir/gravar e clique em **Conectar**.

---

### Passo 3 — Conectar um cliente no terminal (Terminal 2)

Abra um **novo terminal**, entre na mesma pasta e execute:

#### Opção A — Ouvir e gravar (recomendado para demonstração)

```bash
cd Streaming_Audio-Redes
go run ./cmd/client -addr localhost:9090 -play -output output/cliente1.wav
```

#### Opção B — Apenas gravar (sem som no alto-falante)

```bash
cd Streaming_Audio-Redes
go run ./cmd/client -addr localhost:9090 -output output/cliente1.wav
```

#### Opção C — Apenas ouvir (sem gravar arquivo)

```bash
cd Streaming_Audio-Redes
go run ./cmd/client -addr localhost:9090 -play -output -
```

**Logs esperados:**

```
[cliente] conectado a localhost:9090
[cliente] metadados recebidos | codec=pcm container=wav 44100Hz 1ch 16bit | fonte=musica.wav
[cliente] reprodução ao vivo iniciada | 44100Hz 1ch — buffer cresce conforme pacotes chegam
[cliente] pacote #1 recebido (4096 bytes) | estado: reproduzindo/gravando
```

No prompt `> `, teste os comandos:

| Comando | Ação |
|---------|------|
| `pause` ou `p` | Pausa gravação e reprodução local |
| `resume` ou `r` | Retoma gravação e reprodução |
| `status` | Mostra pacotes recebidos e buffer |
| `stop` ou `s` | Encerra o cliente |

---

### Passo 4 — Segundo cliente, opcional (Terminal 3)

Para demonstrar **múltiplos ouvintes** no mesmo fluxo:

```bash
cd Streaming_Audio-Redes
go run ./cmd/client -addr localhost:9090 -play -output output/cliente2.wav
```

No servidor, deve aparecer:

```
[servidor] ouvintes ativos: 2
```

---

### Passo 5 — Verificar gravação (após `stop` no cliente)

Se gravou com `-output output/cliente1.wav`, reproduza o arquivo:

**macOS:**

```bash
afplay output/cliente1.wav
```

**Linux (com `aplay`):**

```bash
aplay output/cliente1.wav
```

---

## Roteiro de demonstração para o professor

Execute nesta ordem e verifique cada item:

| # | O que demonstrar | Como verificar |
|---|------------------|----------------|
| 1 | Servidor iniciado | Log `rádio iniciada` no Terminal 1 |
| 2 | Transmissão em loop | Logs `pacote #N enviado` no servidor |
| 3 | Handshake de metadados | Log `metadados enviados/recebidos` |
| 4 | Dois clientes simultâneos | Log `ouvintes ativos: 2` |
| 5 | Mesmo fluxo para todos | Ambos recebem `pacote #N` ao mesmo tempo |
| 6 | Reprodução ao vivo | Som contínuo com `-play` |
| 7 | Comandos locais | `pause`, `resume`, `stop` no prompt `> ` |
| 8 | Gravação concatenada | Arquivo `output/cliente1.wav` reproduzível |

---

## Problemas comuns

| Erro | Causa | Solução |
|------|-------|---------|
| `go.mod file not found` | Terminal na pasta errada | `cd Streaming_Audio-Redes` antes de `go run` |
| `no such file: assets/musica.wav` | Arquivo de áudio ausente | Coloque um `.wav` em `assets/` ou use `-audio caminho/arquivo.wav` |
| `connection refused` | Servidor não está rodando | Inicie o servidor no Terminal 1 primeiro |
| `address already in use` | Porta 9090 ocupada | Use outra porta: `-addr :9091` no servidor e `-addr localhost:9091` no cliente |
| Sem som com `-play` | Arquivo não é PCM 16-bit | Use `.wav` PCM; MP3 só grava, não reproduz ao vivo ainda |
| Sem captura no modo live | Permissão de microfone negada | macOS: Ajustes → Privacidade → Microfone → permitir Terminal/IDE |
| `dispositivo de entrada não encontrado` | Nome errado em `-input` | Rode `-list-devices` e use parte do nome listado |

---

## Funcionalidades implementadas

### Servidor (`cmd/server`)

- [x] Servidor TCP em Go escutando conexões na porta configurável (`:9090`)
- [x] Aceita **múltiplos clientes simultâneos** (goroutines + mapa thread-safe)
- [x] Leitura contínua de arquivo `.wav` ou `.mp3` em **loop** (`-source file`)
- [x] **Captura ao vivo do microfone** e transmissão em tempo real (`-source live`)
- [x] Listagem de dispositivos de entrada (`-list-devices`)
- [x] Transmissão em blocos de **1 KB** com **pacing em tempo real** (ritmo do áudio)
- [x] **Estúdio do locutor** (`cmd/studio`) — mixagem música + microfone, volumes e seek
- [x] Playlist de faixas WAV com validação de formato
- [x] Atualização de metadados mid-stream ao trocar a fonte no ar
- [x] **Broadcast**: o mesmo pacote de áudio é enviado a todos os ouvintes ao mesmo tempo
- [x] Envio de **metadados JSON** ao conectar (codec, sample rate, canais, bits, fonte)
- [x] Transmissão de **somente PCM** — cabeçalho `RIFF` do arquivo é ignorado pelo servidor
- [x] Detecção automática do formato do arquivo (`internal/server/format.go`)
- [x] Logs de conexão, desconexão, quantidade de ouvintes e envio de pacotes

### Cliente (`cmd/client`)

- [x] Conexão TCP ao servidor e recepção do handshake de metadados
- [x] Gravação do fluxo em arquivo `.wav` com **concatenação contínua**
- [x] Montagem local do cabeçalho WAV a partir dos metadados recebidos
- [x] Gravação **pacote a pacote** com sincronização imediata no disco (`Sync`)
- [x] **Reprodução em tempo real** no alto-falante (`-play`) com buffer limitado (~500 ms)
- [x] Tratamento de **troca de fonte** (metadados mid-stream do estúdio)
- [x] Cliente que entra no meio do loop começa a receber/gravar **a partir daquele ponto**
- [x] Comandos interativos: `pause`, `resume`, `stop` e `status`
- [x] Opção de ouvir sem gravar (`-output -`) ou gravar sem ouvir (sem `-play`)
- [x] Logs de pacotes recebidos, bytes gravados e tamanho do buffer de reprodução

### Protocolo (`internal/protocol`)

- [x] Framing TCP: `[4 bytes tamanho][1 byte tipo][payload]`
- [x] Frame `0x01` — metadados (uma vez por conexão)
- [x] Frame `0x02` — áudio (fluxo contínuo)
- [x] Compatível com entrada tardia de clientes (transmissão contínua, sem reinício por cliente)

### Comportamento de rádio

- [x] Transmissão contínua simulando rádio FM (não é download de arquivo)
- [x] Novo ouvinte sintoniza no ponto atual da transmissão, como rádio real
- [x] Suporte a dois ou mais clientes recebendo o **mesmo fluxo** sincronizado

---

## Possíveis melhorias futuras

### Áudio e transmissão

- [x] **Captura ao vivo** — transmitir áudio do microfone (`-source live`)
- [ ] **Decoder MP3 ao vivo** — reproduzir `.mp3` no alto-falante, não só gravar
- [ ] **Playlist** — alternar entre várias faixas sem reiniciar o servidor
- [ ] **Controle da rádio no servidor** — pausar/retomar a transmissão globalmente
- [ ] **Normalização de volume** — equalizar nível entre faixas diferentes

### Protocolo e redes

- [ ] **UDP + RTP** — protocolo pensado para streaming com menor latência
- [ ] **TLS** — criptografar a conexão TCP
- [ ] **Autenticação** — senha ou token para conectar à rádio
- [ ] **Descoberta na rede local** — broadcast/mDNS para achar a rádio sem IP fixo
- [ ] **Interface web** — cliente no navegador via WebSocket/WebRTC

### Cliente e experiência

- [ ] **Cliente mobile** — app ou página acessível pelo celular
- [ ] **Indicador de buffer** — barra visual do atraso entre recepção e reprodução
- [ ] **Reconexão automática** — retomar sessão após queda de conexão
- [x] **Interface gráfica (GUI)** — `cmd/gui` com botões de conectar/pausar/retomar
- [ ] **Exibir metadados da faixa** — nome, duração, artista (ID3 / tags WAV)

### Servidor e operação

- [ ] **Métricas** — dashboard com ouvintes ativos, bitrate, uptime
- [ ] **Limite de ouvintes** — configurar capacidade máxima
- [ ] **Docker** — empacotar servidor e cliente para deploy simplificado
- [ ] **Testes automatizados** — testes de integração servidor + cliente
- [ ] **CI/CD** — pipeline de build e validação no GitHub

---

## Estrutura do projeto

```
Streaming_Audio-Redes/
├── go.mod / go.sum          # dependências
├── cmd/
│   ├── studio/main.go       # estúdio do locutor (GUI servidor)
│   ├── server/main.go       # servidor simples (file ou live, terminal)
│   ├── client/main.go       # cliente (terminal)
│   └── gui/main.go          # cliente com interface gráfica
├── internal/
│   ├── protocol/            # frames TCP + metadados JSON
│   ├── server/              # rádio, broadcast, arquivo e captura ao vivo
│   └── client/              # conexão, gravação, reprodução (oto)
├── assets/                  # arquivos de áudio (.wav / .mp3)
└── output/                  # gravações dos clientes (gerado em runtime)
```

---

## Referência de flags

### Servidor

| Flag | Padrão | Descrição |
|------|--------|-----------|
| `-addr` | `:9090` | Endereço e porta TCP |
| `-audio` | `assets/musica.wav` | Arquivo transmitido em loop |
| `-chunk-ms` | `20` | Intervalo entre blocos (ms) |

### Cliente

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
| Metadados | `0x01` | Uma vez, ao conectar |
| Áudio | `0x02` | Continuamente (broadcast) |

Exemplo de metadados:

```json
{
  "codec": "pcm",
  "container": "wav",
  "sample_rate": 44100,
  "channels": 1,
  "bits_per_sample": 16,
  "source": "musica.wav"
}
```

O servidor transmite **somente PCM** (sem cabeçalho `RIFF`). O cliente monta o cabeçalho WAV localmente a partir dos metadados e concatena os pacotes recebidos em um único arquivo reproduzível.

