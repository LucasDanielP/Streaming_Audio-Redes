package capture

import (
	"fmt"
	"strings"

	"github.com/gen2brain/malgo"
)

// ListDevices retorna microfones disponíveis no sistema.
func ListDevices() ([]string, error) {
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = ctx.Uninit()
		ctx.Free()
	}()

	devices, err := ctx.Devices(malgo.Capture)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(devices))
	for _, d := range devices {
		label := d.Name()
		if d.IsDefault != 0 {
			label += " (padrão)"
		}
		names = append(names, label)
	}
	return names, nil
}

// FindDevice localiza um dispositivo de captura por substring do nome.
func FindDevice(ctx *malgo.AllocatedContext, nameQuery string) (malgo.DeviceID, string, error) {
	devices, err := ctx.Devices(malgo.Capture)
	if err != nil {
		return malgo.DeviceID{}, "", err
	}

	query := strings.ToLower(strings.TrimSpace(nameQuery))
	for _, d := range devices {
		if strings.Contains(strings.ToLower(d.Name()), query) {
			return d.ID, d.Name(), nil
		}
	}
	return malgo.DeviceID{}, "", fmt.Errorf("dispositivo de entrada não encontrado: %q", nameQuery)
}
