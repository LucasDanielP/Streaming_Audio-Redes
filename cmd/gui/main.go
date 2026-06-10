package main

import (
	"fmt"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"streaming-audio-redes/internal/listener"
	"streaming-audio-redes/internal/ui"
)

type session struct {
	mu       sync.Mutex
	listener *listener.Listener
	running  bool
}

func (s *session) setListener(l *listener.Listener) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listener = l
	s.running = l != nil
}

func (s *session) getListener() *listener.Listener {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.listener
}

func (s *session) isRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

func (s *session) clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listener = nil
	s.running = false
}

func main() {
	a := app.NewWithID("br.ufes.back2past")
	ui.ApplyTheme(a)
	w := a.NewWindow("Back2Past — Rádio")
	w.Resize(ui.GUIWindowSize())

	var sess session

	addrEntry := widget.NewEntry()
	addrEntry.SetText("localhost:9090")
	addrEntry.SetPlaceHolder("localhost:9090")

	outputEntry := widget.NewEntry()
	outputEntry.SetText("output/ouvindo.wav")
	outputEntry.SetPlaceHolder("output/ouvindo.wav")

	playCheck := widget.NewCheck("Reproduzir ao vivo", nil)
	playCheck.SetChecked(true)

	saveCheck := widget.NewCheck("Gravar em arquivo", nil)
	saveCheck.SetChecked(true)

	nowPlayingCaption := ui.MonoCaption("Tocando agora · " + ui.StationGenre)
	trackTitle := canvas.NewText("Sintonize a estação", ui.ColorForeground)
	trackTitle.TextSize = 16
	trackTitle.TextStyle = fyne.TextStyle{Bold: true}

	trackDetail := canvas.NewText("Conecte-se para ouvir a transmissão", ui.ColorMuted)
	trackDetail.TextSize = 11

	elapsedLabel := canvas.NewText("00:00", ui.ColorPrimary)
	elapsedLabel.TextSize = 13
	elapsedLabel.TextStyle = fyne.TextStyle{Monospace: true}

	onAirBadge := ui.NewOnAirBadge()
	vuMeter := ui.NewVuMeter()

	setControlsEnabled := func(connected bool) {
		addrEntry.Disable()
		outputEntry.Disable()
		playCheck.Disable()
		saveCheck.Disable()
		if !connected {
			addrEntry.Enable()
			outputEntry.Enable()
			playCheck.Enable()
			saveCheck.Enable()
		}
	}

	var connectStart time.Time
	var elapsedSec int

	updateDisplay := func() {
		l := sess.getListener()
		if l == nil {
			ui.SetOnAir(onAirBadge, false)
			trackTitle.Text = "Sintonize a estação"
			trackDetail.Text = "Conecte-se para ouvir a transmissão"
			nowPlayingCaption.Text = "Fora do ar"
			elapsedLabel.Text = "00:00"
			vuMeter.SetActive(false)
			return
		}

		st := l.Stats()
		connected := st.State != listener.StateStopped
		playing := st.State == listener.StatePlaying
		ui.SetOnAir(onAirBadge, connected)
		vuMeter.SetActive(playing)

		if st.Source != "" {
			trackTitle.Text = st.Source
			trackDetail.Text = fmt.Sprintf(
				"%s · ~%.1f s gravado · atraso ~%.0f ms",
				st.StateLabel, st.DurationSec, st.LatencyMs,
			)
			nowPlayingCaption.Text = "Tocando agora · " + ui.StationGenre
		} else {
			trackTitle.Text = "Conectando..."
			trackDetail.Text = st.StateLabel
		}

		if playing {
			elapsedSec = int(time.Since(connectStart).Seconds())
		}
		elapsedLabel.Text = fmt.Sprintf("%02d:%02d", elapsedSec/60, elapsedSec%60)

		canvas.Refresh(nowPlayingCaption)
		canvas.Refresh(trackTitle)
		canvas.Refresh(trackDetail)
		canvas.Refresh(elapsedLabel)
	}

	connectBtn := widget.NewButton("▶  Conectar", nil)
	pauseBtn := widget.NewButton("Pausar", nil)
	resumeBtn := widget.NewButton("Retomar", nil)
	stopBtn := widget.NewButton("Desconectar", nil)

	pauseBtn.Disable()
	resumeBtn.Disable()
	stopBtn.Disable()

	connectBtn.OnTapped = func() {
		if sess.isRunning() {
			return
		}

		output := outputEntry.Text
		if !saveCheck.Checked {
			output = "-"
		}

		l := listener.New(addrEntry.Text, output, playCheck.Checked)
		sess.setListener(l)
		connectStart = time.Now()
		elapsedSec = 0
		setControlsEnabled(true)
		pauseBtn.Enable()
		resumeBtn.Enable()
		stopBtn.Enable()
		connectBtn.Disable()
		updateDisplay()

		go func() {
			err := l.RunHeadless()
			fyne.Do(func() {
				sess.clear()
				elapsedSec = 0
				setControlsEnabled(false)
				pauseBtn.Disable()
				resumeBtn.Disable()
				stopBtn.Disable()
				connectBtn.Enable()
				if err != nil {
					trackDetail.Text = "Erro: " + err.Error()
				}
				updateDisplay()
			})
		}()
	}

	pauseBtn.OnTapped = func() {
		if l := sess.getListener(); l != nil {
			l.Pause()
			vuMeter.SetActive(false)
			updateDisplay()
		}
	}

	resumeBtn.OnTapped = func() {
		if l := sess.getListener(); l != nil {
			l.Resume()
			updateDisplay()
		}
	}

	stopBtn.OnTapped = func() {
		if l := sess.getListener(); l != nil {
			l.Stop()
		}
	}

	go func() {
		for range time.Tick(500 * time.Millisecond) {
			if !sess.isRunning() {
				continue
			}
			fyne.Do(updateDisplay)
		}
	}()

	nowPlayingRow := container.NewBorder(
		nil, nil, nil,
		elapsedLabel,
		container.NewVBox(nowPlayingCaption, trackTitle, trackDetail),
	)

	nowPlayingPanel := ui.DisplayPanel(ui.SpacedVBox(4,
		nowPlayingRow,
		vuMeter,
	))

	transport := container.NewGridWithColumns(4,
		connectBtn, pauseBtn, resumeBtn, stopBtn,
	)

	tuneSection := ui.DisplayPanel(ui.SpacedVBox(4,
		ui.MonoCaption("Sintonizar"),
		addrEntry,
		container.NewGridWithColumns(2, playCheck, saveCheck),
		outputEntry,
	))

	header := ui.BrandHeaderRow(
		ui.StationName,
		ui.StationFreq+" FM · "+ui.StationCity,
		false,
		onAirBadge,
	)

	content := ui.SpacedVBox(ui.GUISectionGap,
		header,
		nowPlayingPanel,
		transport,
		tuneSection,
	)

	scroll := container.NewScroll(ui.CardWrap(content))
	scroll.SetMinSize(fyne.NewSize(ui.GUIWindowSize().Width, 0))
	w.SetContent(scroll)
	w.CenterOnScreen()
	w.ShowAndRun()
}
