package vmpas_test

import (
	"os/exec"
	"strings"
	"testing"
)

// TestVMPasHasNoExternalDeps blinda la regla arquitectónica central del
// proyecto: pkg/vmpas (Pascal embebido en Go) debe poder importarse en
// cualquier programa sin arrastrar la rama de la IDE ni dependencias
// externas. En particular tcell, que vive exclusivamente bajo internal/tui
// y cmd/turbo, jamás debe aparecer en el cierre transitivo de imports de
// pkg/vmpas. Si este test falla, alguien introdujo un import accidental que
// contamina a los consumidores de la librería.
func TestVMPasHasNoExternalDeps(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", "github.com/arturoeanton/go-turbo-pascal/pkg/vmpas").Output()
	if err != nil {
		t.Skipf("no se pudo ejecutar 'go list -deps' (toolchain no disponible): %v", err)
	}
	const self = "github.com/arturoeanton/go-turbo-pascal"
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		pkg := strings.TrimSpace(line)
		if pkg == "" {
			continue
		}
		// Paquetes de la stdlib no tienen punto en el primer segmento de ruta.
		first, _, _ := strings.Cut(pkg, "/")
		isStdlib := !strings.Contains(first, ".")
		isSelf := strings.HasPrefix(pkg, self)
		if isStdlib || isSelf {
			continue
		}
		t.Errorf("pkg/vmpas no debe depender de paquetes externos, pero importa %q "+
			"(las dependencias externas como tcell solo pueden vivir en la rama de la IDE)", pkg)
	}
}
