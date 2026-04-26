# Dynamic Background for Sway - Spezifikation

## 1. Projektziel

Erzeugung eines dynamischen Hintergrundbildes für den Sway Wayland Compositor, das in Echtzeit Systeminformationen anzeigt. Die angezeigten Informationen sind frei definierbar und konfigurierbar über eine API.

## 2. Kernanforderungen

### 2.1 Hintergrund-Rendering
- Vollbild-Hintergrund für Sway (Wayland-kompatibel)
- Echtzeit-Aktualisierung der angezeigten Informationen
- Unterstützung für statische Bilder, Farbverläufe oder generierte Grafiken als Basis
- Performance-optimiert (geringe CPU/GPU-Auslastung)

### 2.2 Systeminformationen
Folgende Informationen sollen abrufbar und anzeigbar sein (erweiterbar):
- CPU-Auslastung (gesamt & pro Kern)
- RAM-Nutzung
- Festplatten-Speicher
- Netzwerk-Traffic (Up/Down)
- Batterie-Status (Laptop)
- System-Uptime
- Datum & Uhrzeit
- Wetter-Daten (über externe API)
- Custom-Scripts/Commands

### 2.3 API-Design
- RESTful API oder WebSocket für Echtzeit-Updates
- Konfiguration der angezeigten Widgets/Informationen
- Layout-Definition (Position, Größe, Stil)
- Hot-Reload der Konfiguration ohne Neustart

## 3. Architektur

### 3.1 Rendering-Strategie
**Wayland Layer Shell Protocol** – eigenes Surface als Background-Layer.
- Kein Flackern, kein swaybg nötig
- Direktes Rendering via Cairo auf Shared-Memory-Buffer
- Dirty-Rect-Optimierung: Nur geänderte Widget-Bereiche neu rendern

### 3.2 Refresh-Strategie (Event-Driven, kein Fixed Framerate)

| Widget       | Intervall | Grund                |
|-------------|-----------|----------------------|
| CPU         | 1-2s      | Schnell wechselnd    |
| RAM         | 5s        | Langsam wechselnd    |
| Disk        | 30s       | Sehr stabil          |
| Clock       | 1s        | Sichtbar             |
| Network     | 2-5s      | Mittel               |
| Battery     | 10s       | Langsam              |
| Weather     | 15-30min  | Externer API-Call    |
| Custom      | Config    | User-defined         |

### 3.3 Diagramm

```
┌─────────────────────────────────────────────┐
│         Wayland Layer Shell Surface          │
│         (Background Layer, Fullscreen)       │
├─────────────────────────────────────────────┤
│           Cairo Rendering Engine             │
│  (Persistent Surface, Dirty-Rect Updates)    │
├─────────────────────────────────────────────┤
│         Widget System                        │
│  ┌─────┐ ┌─────┐ ┌─────┐ ┌─────┐            │
│  │ CPU │ │ RAM │ │ NET │ │ ... │            │
│  │ 1s  │ │ 5s  │ │ 2s  │ │     │            │
│  └─────┘ └─────┘ └─────┘ └─────┘            │
├─────────────────────────────────────────────┤
│      Timer Scheduler (per Widget)            │
│  (Event-driven, kein fixed framerate)        │
├─────────────────────────────────────────────┤
│         Data Collector Layer                 │
│  (/proc, /sys, netlink, externe APIs)        │
├─────────────────────────────────────────────┤
│         Configuration API                    │
│  (HTTP + WebSocket Server)                   │
└─────────────────────────────────────────────┘
```

## 4. Technologie-Stack

### 4.1 Sprache
- Go (da Projekt in GOPATH)

### 4.2 Wayland-Integration
- `github.com/riverwall/go-wayland` oder `github.com/fsouza/go-wayland`
- Layer Shell Protocol (`zwlr_layer_shell_v1`) für Background-Layer
- Shared Memory (`wl_shm`) für Buffer
- Cairo für Rendering auf SHM-Buffer

### 4.3 Rendering
- Option A: Cairo (`github.com/ungerik/go-cairo`)
- Option B: OpenGL über GLFW
- Option C: SVG-basiert mit Template-Engine

### 4.4 API
- HTTP Server (stdlib `net/http`)
- WebSocket für Echtzeit-Updates (`gorilla/websocket`)
- JSON für Konfiguration

## 5. Konfigurationsformat

```yaml
# config.yaml
background:
  type: gradient  # image, gradient, solid, animated
  colors: ["#1a1a2e", "#16213e", "#0f3460"]
  image_path: "/path/to/image.png"

layout:
  - widget: cpu
    position: { x: 20, y: 20 }
    size: { width: 200, height: 100 }
    style:
      font: "Monospace 12"
      color: "#ffffff"
      background: "rgba(0,0,0,0.5)"
      
  - widget: ram
    position: { x: 240, y: 20 }
    size: { width: 200, height: 100 }
    
  - widget: custom
    command: "uptime -p"
    interval: 60s
    position: { x: 20, y: 140 }

api:
  enabled: true
  port: 8080
  websocket: true
```

## 6. API-Endpunkte

### 6.1 REST API
- `GET /api/v1/widgets` - Liste aller Widgets
- `POST /api/v1/widgets` - Neues Widget hinzufügen
- `PUT /api/v1/widgets/{id}` - Widget aktualisieren
- `DELETE /api/v1/widgets/{id}` - Widget entfernen
- `GET /api/v1/config` - Gesamte Konfiguration
- `PUT /api/v1/config` - Konfiguration aktualisieren
- `GET /api/v1/system` - Systeminformationen (Live-Daten)

### 6.2 WebSocket
- Endpoint: `ws://localhost:8080/ws`
- Events: `widget_update`, `config_change`, `system_info`

## 7. Widget-Typen

### 7.1 Built-in Widgets
- `cpu` - CPU-Auslastung
- `memory` / `ram` - RAM-Nutzung
- `disk` - Festplatten-Speicher
- `network` - Netzwerk-Traffic
- `battery` - Batterie-Status
- `clock` - Datum & Uhrzeit
- `uptime` - System-Uptime
- `temperature` - CPU/GPU-Temperaturen

### 7.2 Custom Widgets
- Shell-Commands
- Script-Ausgabe
- Externe API-Calls

## 8. Wayland-Integration

### 8.1 Layer Shell Protocol
- Surface wird als `layer: background` registriert
- Anchor: top, bottom, left, right (Fullscreen)
- Keyboard-Interactivity: none (kein Fokus, keine Input-Events)
- Multi-Monitor: Pro Output ein eigenes Surface

### 8.2 Rendering-Pipeline
1. Wayland Display connect
2. Compositor + Layer Shell Registry binden
3. Pro Output: Layer Surface erstellen
4. SHM Pool + Buffer allozieren
5. Cairo Surface auf SHM-Buffer mappen
6. Render → Buffer commit → Frame-Callback

### 8.3 Dirty-Rect Optimierung
- Jedes Widget trackt ob es "dirty" ist
- Nur dirty Widgets werden neu gezeichnet
- Union aller dirty Rects → minimaler Redraw-Bereich
- Frame-Callback signalisiert wann nächster Frame safe ist

## 9. Projektstruktur

```
dynamic_background/
├── cmd/
│   └── dynamic_background/
│       └── main.go
├── internal/
│   ├── api/
│   │   ├── server.go
│   │   ├── handlers.go
│   │   └── websocket.go
│   ├── config/
│   │   ├── config.go
│   │   └── loader.go
│   ├── renderer/
│   │   ├── renderer.go
│   │   └── widgets.go
│   ├── collectors/
│   │   ├── cpu.go
│   │   ├── memory.go
│   │   ├── network.go
│   │   ├── disk.go
│   │   ├── battery.go
│   │   └── custom.go
│   └── wayland/
│       └── surface.go
├── pkg/
│   └── widget/
│       └── types.go
├── config/
│   └── default.yaml
├── go.mod
└── README.md
```

## 10. Meilensteine

1. **MVP**: Wayland Surface + Cairo Rendering + Clock Widget (Fullscreen, 1 Monitor)
2. **Widgets**: CPU + RAM + Dirty-Rect Optimization + per-Widget Timer
3. **API**: HTTP API für Konfiguration + Hot-Reload
4. **Multi-Monitor**: Pro Output eigenes Surface, korrekte DPI/Scale
5. **WebSocket**: Echtzeit-Updates + Custom Scripts
6. **Theming**: Vollständige Theming-Unterstützung + Animationen

## 12. Entwicklungsprozess & Wissenstransfer

### 12.1 TDD Workflow (Strict)
1. **Test schreiben** (rot)
2. **Code implementieren** (grün)
3. **Refactor** (sauber)
4. **Git Commit** mit Logbuch-Eintrag
5. Repeat

### 12.2 Commit-Logbuch Format
Jeder Commit enthält:
- **Warum** wurde der Code geschrieben?
- **Wozu** dient er?
- **Was passiert** wenn er fehlt?

```
feat: add CPU collector with sysfs parsing

WHY: CPU-Auslastung ist Kern-Widget, muss effizient aus /proc/stat gelesen werden
WHAT: Parser der Jiffies differenziert und Prozent berechnet
IMPACT: Ohne diesen Code gibt es kein CPU-Widget, Background zeigt keine CPU-Info
```

### 12.3 Clean Architecture + DDD

```
┌─────────────────────────────────────────────┐
│              Interfaces (API/CLI)            │
│         (Adapter nach außen)                 │
├─────────────────────────────────────────────┤
│           Application Layer                  │
│      (Use Cases, Orchestrierung)             │
├─────────────────────────────────────────────┤
│            Domain Layer                      │
│   (Entities, Value Objects, Interfaces)      │
│         ← KEINE Dependencies →              │
├─────────────────────────────────────────────┤
│         Infrastructure Layer                 │
│  (Wayland, Cairo, /proc, HTTP Server)        │
└─────────────────────────────────────────────┘
```

**Dependency Rule**: Dependencies zeigen NUR nach innen zum Domain Layer.

### 12.4 Dokumentation

| Typ | Ort | Inhalt |
|-----|-----|--------|
| ADRs | `docs/decisions/` | Architektur-Entscheidungen mit Kontext |
| Code-Docs | `docs/` | Architecture, Widget Guide, Wayland Primer |
| Inline | Im Code | Warum, nicht Was |
| CLI | `dynamic-background explain` | Executable Documentation |

### 12.5 SOLID Prinzipien

- **S**: Ein Widget = eine Responsibility
- **O**: Neue Widgets via Interface, nicht Modification
- **L**: Collector-Implementierungen austauschbar
- **I**: Schlanke Interfaces (Collector, Renderer, Widget)
- **D**: Domain definiert Interfaces, Infrastructure implementiert

### 12.6 Unit Tests
- **Coverage-Ziel**: >80%
- **Jede öffentliche Methode** hat mindestens einen Test
- **Mocks** für Wayland, /proc, externe APIs
- **Table-Driven Tests** für Collector-Parser

## 13. Offene Fragen

- Animationen unterstützt (z.B. animierte Gradienten, Übergänge)?
- Soll eine GUI zur Konfiguration bereitgestellt werden?
- Fallback für nicht-Wayland Sessions (X11)?
