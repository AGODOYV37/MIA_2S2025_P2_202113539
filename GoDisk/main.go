package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/catalog"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/commands"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/ext2"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/ext3"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/mount"
	"github.com/AGODOYV37/MIA_2S2025_P2_202113539/internal/reports"
	u "github.com/AGODOYV37/MIA_2S2025_P2_202113539/pkg"
)

// ---------------------- Infra de aplicación ----------------------

type App struct {
	reg       *mount.Registry
	svc       *mount.Service
	formatter *ext2.Formatter
	mu        sync.Mutex // serializa ejecuciones (estado compartido)
}

func NewApp() *App {
	reg := mount.NewRegistry()
	svc := mount.NewService(reg)
	formatter := ext2.NewFormatter(reg)
	_ = reg.RehydrateFromCatalog()
	return &App{reg: reg, svc: svc, formatter: formatter}

}

func (a *App) handleReportMBR(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if id == "" {
		writeJSONError(w, http.StatusBadRequest, "mbr: id requerido")
		return
	}
	if _, ok := a.reg.GetByID(id); !ok {
		writeJSONError(w, http.StatusNotFound, fmt.Sprintf("mbr: id %q no está montado", id))
		return
	}

	rep, err := reports.BuildMBR(a.reg, id)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, rep)
}

func (a *App) handleReportDisk(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	rep, err := reports.BuildDisk(a.reg, id)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(rep)
}

func (a *App) handleReportInode(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	id := strings.TrimSpace(q.Get("id"))
	ruta := strings.TrimSpace(q.Get("ruta")) // puede venir vacío
	rep, err := reports.BuildInode(a.reg, id, ruta)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(rep)
}

func (a *App) handleReportInodes(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	id := strings.TrimSpace(q.Get("id"))
	max := 0
	if s := strings.TrimSpace(q.Get("max")); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			max = n
		}
	}
	rep, err := reports.BuildInodes(a.reg, id, max)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(rep)
}

// GET /api/reports/block?id=XXXX
func (a *App) handleReportBlock(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")

	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "id requerido"})
		return
	}

	rep, err := reports.BuildBlock(a.reg, id)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(rep)
}

func (a *App) handleReportBmInode(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")

	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if id == "" {
		http.Error(w, "id requerido", http.StatusBadRequest)
		return
	}

	txt, err := reports.BuildBmInodeText(a.reg, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_, _ = w.Write([]byte(txt))
}

func (a *App) handleReportBmBlock(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")

	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if id == "" {
		http.Error(w, "id requerido", http.StatusBadRequest)
		return
	}

	txt, err := reports.BuildBmBlockText(a.reg, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_, _ = w.Write([]byte(txt))
}

func (a *App) handleReportTree(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "solo GET", http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if format == "" {
		format = "json"
	}

	if id == "" {
		http.Error(w, `falta query ?id=`, http.StatusBadRequest)
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	rep, err := reports.BuildTree(a.reg, id)
	if err != nil {
		http.Error(w, "tree: "+err.Error(), http.StatusBadRequest)
		return
	}

	switch format {
	case "json":
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(rep)
	case "html":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		html := reports.RenderHTMLTree(rep)
		_, _ = io.WriteString(w, html)
	default:
		http.Error(w, "format debe ser json|html", http.StatusBadRequest)
	}
}

func (a *App) handleReportSB(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	rep, err := reports.BuildSB(a.reg, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	// deshabilitamos caches agresivos del navegador
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(rep)
}

func (a *App) handleReportFile(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")

	// AHORA: parámetro principal = "ruta"
	ruta := r.URL.Query().Get("ruta")
	// compatibilidad (opcional)
	if ruta == "" {
		ruta = r.URL.Query().Get("path_file")
	}
	if ruta == "" {
		ruta = r.URL.Query().Get("path_file_ls")
	}
	if id == "" || ruta == "" {
		http.Error(w, "rep file: se requieren id y ruta", http.StatusBadRequest)
		return
	}

	data, err := reports.BuildFile(a.reg, id, ruta) // antes usabas pathFile
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	name := filepath.Base(ruta)
	if name == "" || name == "/" || name == "." {
		name = "file_" + id + ".txt"
	}
	if filepath.Ext(name) == "" {
		name += ".txt"
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", name))
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func writeJSONError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func (app *App) handleReportLS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "solo GET", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if id == "" {
		http.Error(w, "rep ls: -id requerido", http.StatusBadRequest)
		return
	}
	ruta := strings.TrimSpace(r.URL.Query().Get("ruta"))
	if ruta == "" {
		ruta = "/"
	}

	rep, err := reports.BuildLS(app.reg, id, ruta)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, rep)
}

func captureOutput(fn func()) (string, string) {
	origOut, origErr := os.Stdout, os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout, os.Stderr = wOut, wErr

	outCh := make(chan string)
	errCh := make(chan string)

	go func() {
		var b bytes.Buffer
		_, _ = io.Copy(&b, rOut)
		outCh <- b.String()
	}()
	go func() {
		var b bytes.Buffer
		_, _ = io.Copy(&b, rErr)
		errCh <- b.String()
	}()

	fn() // ejecuta la lógica que imprime

	_ = wOut.Close()
	_ = wErr.Close()
	os.Stdout, os.Stderr = origOut, origErr

	stdout := <-outCh
	stderr := <-errCh
	return stdout, stderr
}

// ProcessLine: ejecuta UNA línea de comando y devuelve TODO lo impreso.
func (a *App) ProcessLine(line string) string {
	line = strings.TrimSpace(line)
	line = strings.TrimLeft(line, "\uFEFF")
	if line == "" || strings.HasPrefix(line, "#") {
		return ""
	}

	if strings.EqualFold(line, "exit") {
		return "Saliendo (modo CLI); ignorado en modo HTTP.\n"
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	stdout, stderr := captureOutput(func() {

		tokens := u.Tokeniza(line)
		if len(tokens) == 0 {
			return
		}
		command := strings.ToLower(tokens[0])
		args := u.NormalizaFlags(tokens[1:])

		switch command {
		case "mkdisk":
			fs := flag.NewFlagSet("mkdisk", flag.ContinueOnError)
			fs.SetOutput(io.Discard)
			size := fs.Int("size", 0, "Tamaño del disco.")
			unit := fs.String("unit", "m", "Unidad del tamaño (k/m).")
			fit := fs.String("fit", "ff", "Tipo de ajuste (bf/ff/wf).")
			path := fs.String("path", "", "Ruta del disco a crear.")
			if err := fs.Parse(args); err != nil {
				fmt.Println("Error:", err)
				return
			}

			if *path == "" {
				fmt.Println("Error: el parámetro -path es obligatorio para mkdisk.")
				return
			}
			if *size <= 0 {
				fmt.Println("Error: el parámetro -size es obligatorio y debe ser positivo.")
				return
			}
			if err := commands.ExecuteMkdisk(*size, *unit, *fit, *path); err != nil {
				fmt.Println("Error:", err)
				return
			}

			fmt.Println("Disco creado exitosamente en:", *path)
			_ = catalog.Add(*path)
			_ = a.reg.RehydrateFromCatalog()

		case "rmdisk":
			fs := flag.NewFlagSet("rmdisk", flag.ContinueOnError)
			fs.SetOutput(io.Discard)
			path := fs.String("path", "", "Ruta del disco a eliminar.")
			if err := fs.Parse(args); err != nil {
				fmt.Println("Error:", err)
				return
			}
			if *path == "" {
				fmt.Println("Error: el parámetro -path es obligatorio para rmdisk.")
				return
			}

			removed := commands.ExecuteRmdisk(*path)
			if removed {
				_ = catalog.Remove(*path)
				// purga estado en RAM inmediatamente
				count := a.reg.PurgeDisk(*path)
				fmt.Printf("rmdisk: purgado %d montaje(s) de RAM\n", count)
				_ = a.reg.RehydrateFromCatalog()
			}

		case "fdisk":
			fs := flag.NewFlagSet("fdisk", flag.ContinueOnError)
			fs.SetOutput(io.Discard)
			size := fs.Int64("size", 0, "Tamaño de la partición al CREAR.")
			path := fs.String("path", "", "Ruta del disco (.mia/.dk).")
			name := fs.String("name", "", "Nombre de la partición (único por disco).")
			unit := fs.String("unit", "k", "Unidad (b/k/m). Aplica para -size y -add.")
			typeStr := fs.String("type", "p", "Tipo de partición al CREAR (p/e/l).")
			fit := fs.String("fit", "wf", "Ajuste al CREAR (bf/ff/wf).")
			del := fs.String("delete", "", "Elimina partición por nombre: fast|full.")
			add := fs.Int64("add", 0, "Agrega(+) o quita(-) espacio a la partición.")
			if err := fs.Parse(args); err != nil {
				fmt.Println("Error:", err)
				return
			}
			if strings.TrimSpace(*path) == "" {
				fmt.Println("Error: -path es obligatorio.")
				return
			}
			// DELETE
			if strings.TrimSpace(*del) != "" {
				if strings.TrimSpace(*name) == "" {
					fmt.Println("Error: -name es obligatorio para -delete.")
					return
				}
				if err := commands.ExecuteFdisk(commands.FdiskOptions{
					Path: *path, Name: *name, Delete: strings.ToLower(*del),
				}); err != nil {
					fmt.Println("Error:", err)
					return
				}
				_ = catalog.Add(*path)
				_ = a.reg.RehydrateFromCatalog()
				return
			}
			//  ADD (crece/encoge)
			if *add != 0 {
				if strings.TrimSpace(*name) == "" {
					fmt.Println("Error: -name es obligatorio para -add.")
					return
				}
				if err := commands.ExecuteFdisk(commands.FdiskOptions{
					Path: *path, Name: *name, Unit: strings.ToLower(*unit), Add: *add,
				}); err != nil {
					fmt.Println("Error:", err)
					return
				}
				_ = catalog.Add(*path)
				_ = a.reg.RehydrateFromCatalog()
				return
			}
			// CREATE
			if strings.TrimSpace(*name) == "" || *size <= 0 {
				fmt.Println("Error: para CREAR se requieren -name y -size > 0.")
				return
			}
			if err := commands.ExecuteFdisk(commands.FdiskOptions{
				Path: *path, Name: *name, Unit: strings.ToLower(*unit),
				Type: strings.ToLower(*typeStr), Fit: strings.ToLower(*fit), Size: *size,
			}); err != nil {
				fmt.Println("Error:", err)
				return
			}
			_ = catalog.Add(*path)
			_ = a.reg.RehydrateFromCatalog()

		case "mount":
			fs := flag.NewFlagSet("mount", flag.ContinueOnError)
			fs.SetOutput(io.Discard)
			path := fs.String("path", "", "Ruta del disco (.mia)")
			name := fs.String("name", "", "Nombre de la partición (primaria)")
			if err := fs.Parse(args); err != nil {
				fmt.Println("Error:", err)
				return
			}
			if *path == "" || *name == "" {
				fmt.Println("Error: -path y -name son obligatorios para mount.")
				return
			}
			if id, err := a.svc.Mount(*path, *name); err != nil {
				fmt.Println("Error:", err)
			} else {
				fmt.Println("Particion montada, ID=", id)
				_ = catalog.Add(*path)
				_ = a.reg.RehydrateFromCatalog()
			}

		case "mounted":
			fs := flag.NewFlagSet("mounted", flag.ContinueOnError)
			fs.SetOutput(io.Discard)
			asTable := fs.Bool("table", false, "Mostrar en tabla")
			asJSON := fs.Bool("json", false, "Mostrar en JSON")
			if err := fs.Parse(args); err != nil {
				fmt.Println("Error:", err)
				return
			}
			if *asJSON {
				if views, err := a.reg.MountedJSON(); err != nil {
					fmt.Println("(sin particiones montadas)")
				} else {
					b, _ := json.MarshalIndent(views, "", "  ")
					fmt.Println(string(b))
				}
				return
			}
			if *asTable {
				if table, err := a.reg.MountedTable(); err != nil {
					fmt.Println("(sin particiones montadas)")
				} else {
					fmt.Print(table)
				}
				return
			}
			if line, err := a.reg.MountedPlain(); err != nil {
				fmt.Println("(sin particiones montadas)")
			} else {
				fmt.Println(line)
			}

		case "unmount":
			fs := flag.NewFlagSet("unmount", flag.ContinueOnError)
			fs.SetOutput(io.Discard)
			id := fs.String("id", "", "ID de partición montada (p.ej. 39A1)")
			path := fs.String("path", "", "Ruta del disco (.mia)")
			name := fs.String("name", "", "Nombre de la partición (primaria)")
			if err := fs.Parse(args); err != nil {
				fmt.Println("Error:", err)
				return
			}
			switch {
			case strings.TrimSpace(*id) != "":
				if err := a.svc.UnmountByID(*id); err != nil {
					if mount.IsIDNotFound(err) {
						fmt.Printf("Error: ID %q no está montado.\n", *id)
					} else {
						fmt.Println("Error:", err)
					}
				} else {
					fmt.Printf("Desmontado ID=%s\n", *id)
				}
			case strings.TrimSpace(*path) != "" && strings.TrimSpace(*name) != "":
				if err := a.svc.UnmountByPathName(*path, *name); err != nil {
					switch {
					case mount.IsPartitionNotFound(err):
						fmt.Printf("Error: la partición %q no está montada en %s.\n", *name, *path)
					default:
						fmt.Println("Error:", err)
					}
				} else {
					fmt.Printf("Desmontada %q en %s\n", *name, *path)
				}
			default:
				fmt.Println("uso: unmount -id=<ID>  |  unmount -path=\"/ruta/d1.mia\" -name=\"Part1\"")
			}
			return

		case "mkfs":
			fs := flag.NewFlagSet("mkfs", flag.ContinueOnError)
			fs.SetOutput(io.Discard)
			id := fs.String("id", "", "ID de partición montada (p.ej. 39A1)")
			typ := fs.String("type", "full", "Tipo de formateo (solo 'full')")
			fstype := fs.String("fs", "ext2", "Sistema de archivos: ext2|ext3 (default ext2)")
			if err := fs.Parse(args); err != nil {
				fmt.Println("Error:", err)
				return
			}
			if strings.TrimSpace(*id) == "" {
				fmt.Println("uso: mkfs -id=<ID> [-type=full] [-fs=ext2|ext3]")
				return
			}
			if strings.ToLower(strings.TrimSpace(*typ)) != "full" {
				fmt.Println("Aviso: solo se implementa -type=full; se usará full.")
			}

			switch strings.ToLower(strings.TrimSpace(*fstype)) {
			case "ext3":
				// nuevo formateador EXT3
				if err := ext3.NewFormatter(a.reg).MkfsFull(*id); err != nil {
					fmt.Println("Error:", err)
					return
				}
				fmt.Println("mkfs: formateo EXT3 completado en", *id)
			default:
				// ext2 por defecto
				if err := a.formatter.MkfsFull(*id); err != nil {
					fmt.Println("Error:", err)
					return
				}
				fmt.Println("mkfs: formateo EXT2 completado en", *id)
			}

		case "login":
			_ = commands.CmdLogin(a.reg, args)
		case "logout":
			_ = commands.CmdLogout(args)
		case "mkgrp":
			_ = commands.CmdMkgrp(a.reg, args)
		case "rmgrp":
			_ = commands.CmdRmgrp(a.reg, args)
		case "mkusr":
			_ = commands.CmdMkusr(a.reg, args)
		case "rmusr":
			_ = commands.CmdRmusr(a.reg, args)
		case "chgrp":
			_ = commands.CmdChgrp(a.reg, args)
		case "mkfile":
			_ = commands.CmdMkfile(a.reg, args)
		case "mkdir":
			_ = commands.CmdMkdir(a.reg, args)
		case "cat":
			_ = commands.CmdCat(a.reg, args)
		case "rep":
			_ = commands.CmdRep(a.reg, args)
		case "remove":
			_ = commands.CmdRemove(a.reg, args)
		case "edit":
			_ = commands.CmdEdit(a.reg, args)

		default:
			fmt.Printf("Comando '%s' no reconocido.\n", command)
		}
		// ======= FIN switch =======
	})

	if strings.TrimSpace(stderr) == "" {
		return stdout
	}
	if strings.TrimSpace(stdout) == "" {
		return stderr
	}
	return stdout + "\n" + stderr
}

// ---------------------- Handlers HTTP ----------------------

type ExecReq struct {
	Script string `json:"script"`
}
type ExecRes struct {
	Output string `json:"output"`
}

func (a *App) handleExec(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if rec := recover(); rec != nil {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"output": fmt.Sprintf("panic: %v", rec),
			})
		}
	}()

	var req ExecReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"output":"json inválido"}`, http.StatusBadRequest)
		return
	}
	lines := strings.Split(req.Script, "\n")
	var outs []string
	for _, raw := range lines {
		ln := strings.TrimSpace(raw)
		ln = strings.TrimLeft(ln, "\uFEFF")
		if ln == "" || strings.HasPrefix(ln, "#") {
			continue
		}
		outs = append(outs, a.ProcessLine(ln))
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ExecRes{Output: strings.Join(outs, "\n")})
}

func (a *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// ---------------------- Arranques ----------------------

func runHTTP(address string) error {
	app := NewApp()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/exec", app.handleExec)
	mux.HandleFunc("/api/health", app.handleHealth)
	mux.HandleFunc("/api/reports/mbr", app.handleReportMBR)
	mux.HandleFunc("/api/reports/disk", app.handleReportDisk)
	mux.HandleFunc("/api/reports/inode", app.handleReportInode)
	mux.HandleFunc("/api/reports/inodes", app.handleReportInodes)
	mux.HandleFunc("/api/reports/block", app.handleReportBlock)
	mux.HandleFunc("/api/reports/bm_inode", app.handleReportBmInode)
	mux.HandleFunc("/api/reports/bm_block", app.handleReportBmBlock)
	mux.HandleFunc("/api/reports/tree", app.handleReportTree)
	mux.HandleFunc("/api/reports/sb", app.handleReportSB)
	mux.HandleFunc("/api/reports/file", app.handleReportFile)
	mux.HandleFunc("/api/reports/ls", app.handleReportLS)

	fmt.Println("HTTP API escuchando en", address)
	return http.ListenAndServe(address, mux)
}

func runCLI() {
	app := NewApp()
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		line := scanner.Text()
		if strings.EqualFold(line, "exit") {
			fmt.Println("Saliendo...")
			break
		}
		out := app.ProcessLine(line)
		if strings.TrimSpace(out) != "" {
			fmt.Print(out)
			if !strings.HasSuffix(out, "\n") {
				fmt.Println()
			}
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "Error leyendo la entrada:", err)
	}
}

// ---------------------- main ----------------------

func main() {
	httpAddr := flag.String("http", "", "inicia API HTTP en esta dirección (ej.: ':8080')")
	flag.Parse()

	if strings.TrimSpace(*httpAddr) != "" {
		if err := runHTTP(*httpAddr); err != nil {
			fmt.Fprintln(os.Stderr, "Error servidor HTTP:", err)
			os.Exit(1)
		}
		return
	}
	// por defecto: modo CLI
	runCLI()
}
