package shell

import (
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"

	"github.com/getlantern/systray"
)

const serverURL = "http://127.0.0.1:8090"
const settingsURL = serverURL + "/settings"

func generateIcon() []byte {
	const size = 16
	const bpp = 32

	xorSize := size * size * 4
	andRowBytes := ((size + 31) / 32) * 4
	andSize := size * andRowBytes

	bmpDataSize := 40 + xorSize + andSize
	icoDataSize := 6 + 16 + bmpDataSize

	buf := make([]byte, icoDataSize)
	pos := 0

	buf[pos] = 0; pos++
	buf[pos] = 0; pos++
	buf[pos] = 1; pos++
	buf[pos] = 0; pos++
	buf[pos] = 1; pos++
	buf[pos] = 0; pos++

	buf[pos] = size; pos++
	buf[pos] = size; pos++
	buf[pos] = 0; pos++
	buf[pos] = 0; pos++
	binary.LittleEndian.PutUint16(buf[pos:], 1); pos += 2
	binary.LittleEndian.PutUint16(buf[pos:], uint16(bpp)); pos += 2
	binary.LittleEndian.PutUint32(buf[pos:], uint32(bmpDataSize)); pos += 4
	binary.LittleEndian.PutUint32(buf[pos:], uint32(6+16)); pos += 4

	binary.LittleEndian.PutUint32(buf[pos:], 40); pos += 4
	binary.LittleEndian.PutUint32(buf[pos:], uint32(size)); pos += 4
	binary.LittleEndian.PutUint32(buf[pos:], uint32(size*2)); pos += 4
	binary.LittleEndian.PutUint16(buf[pos:], 1); pos += 2
	binary.LittleEndian.PutUint16(buf[pos:], uint16(bpp)); pos += 2
	binary.LittleEndian.PutUint32(buf[pos:], 0); pos += 4
	binary.LittleEndian.PutUint32(buf[pos:], 0); pos += 4
	binary.LittleEndian.PutUint32(buf[pos:], 0); pos += 4
	binary.LittleEndian.PutUint32(buf[pos:], 0); pos += 4
	binary.LittleEndian.PutUint32(buf[pos:], 0); pos += 4
	binary.LittleEndian.PutUint32(buf[pos:], 0); pos += 4

	color := [4]byte{0x20, 0x80, 0xFF, 0xFF}
	for y := size - 1; y >= 0; y-- {
		for x := 0; x < size; x++ {
			copy(buf[pos:], color[:])
			pos += 4
		}
	}

	for i := 0; i < andSize; i++ {
		buf[pos] = 0
		pos++
	}

	return buf
}

func openBrowser(url string) error {
	return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
}

func StartTray(stopServerFn func()) {
	onReady := func() {
		systray.SetIcon(generateIcon())
		systray.SetTooltip("Fileoteca")

		mOpen := systray.AddMenuItem("Abrir Fileoteca", "Abrir en el navegador")
		go func() {
			for range mOpen.ClickedCh {
				if err := openBrowser(serverURL); err != nil {
					fmt.Fprintf(os.Stderr, "error abriendo navegador: %v\n", err)
				}
			}
		}()

		systray.AddSeparator()

		mConfig := systray.AddMenuItem("Configurar", "Abrir configuración")
		go func() {
			for range mConfig.ClickedCh {
				if err := openBrowser(settingsURL); err != nil {
					fmt.Fprintf(os.Stderr, "error abriendo configuración: %v\n", err)
				}
			}
		}()

		systray.AddSeparator()

		mQuit := systray.AddMenuItem("Cerrar", "Cerrar Fileoteca")
		go func() {
			<-mQuit.ClickedCh
			stopServerFn()
			systray.Quit()
		}()
	}

	onExit := func() {}

	systray.Run(onReady, onExit)
}
