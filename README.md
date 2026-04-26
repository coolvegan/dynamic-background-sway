# Dynamic Background

Ein dynamischer Hintergrund für Sway (Wayland) der Systeminformationen in Echtzeit anzeigt.

## Schnellstart

```bash
# Config erstellen
cp config/example.yaml config.yaml

# Starten
go run cmd/dynamic-background/main.go -config config.yaml
```

## Was ist das?

Dieses Projekt erzeugt einen **dynamischen Desktop-Hintergrund** der Systeminformationen anzeigt:

- CPU-Auslastung
- RAM-Nutzung
- Festplatten-Speicher
- Netzwerk-Traffic
- Batterie-Status
- Uhrzeit
- Custom Scripts (beliebige Commands)

### Besonderheiten

- **Per-Widget Timer**: CPU updated jede Sekunde, Disk nur alle 30s → spart Ressourcen
- **Dirty-Rect Rendering**: Nur geänderte Bereiche werden neu gezeichnet
- **Hot-Reload**: Config-Änderungen ohne Neustart via API
- **TDD**: Jede Komponente ist durch Tests abgesichert

## Architektur

```
┌─────────────────────────────────────────────────────────┐
│                    cmd/main.go                          │
│              (Bootstrap & Entry Point)                   │
├─────────────────────────────────────────────────────────┤
│              Application Layer                          │
│  ┌─────────────┐  ┌──────────┐  ┌──────────────────┐   │
│  │ Orchestrator├─►│Scheduler │  │ WidgetManager    │   │
│  │ (Dirigent)  │  │ (Timer)  │  │ (Collector→Widget│   │
│  └──────┬──────┘  └──────────┘  └────────┬─────────┘   │
│         │                                │             │
├─────────┼────────────────────────────────┼─────────────┤
│         │         Domain Layer           │             │
│         │  Widget │ Config │ Collector   │             │
│         │                                │             │
├─────────┼────────────────────────────────┼─────────────┤
│         │     Infrastructure Layer       │             │
│  ┌──────▼──────┐  ┌──────────┐  ┌───────▼─────────┐   │
│  │ Collectors  │  │ Config   │  │ Renderer        │   │
│  │ CPU/Mem/Disk│  │ Loader   │  │ (Interface)     │   │
│  │ Net/Clock/  │  │ (YAML)   │  │                 │   │
│  │ Batt/Custom │  │          │  │ Mock (testing)  │   │
│  └─────────────┘  └──────────┘  │ Image (PNG)     │   │
│                                 │ Cairo (future)  │   │
│                                 └─────────────────┘   │
├─────────────────────────────────────────────────────────┤
│              Interfaces Layer                           │
│  ┌──────────────┐  ┌─────────────────────────────────┐ │
│  │ REST API     │  │ WebSocket                       │ │
│  │ /api/v1/...  │  │ /api/v1/ws                      │ │
│  └──────────────┘  └─────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
```

### Schichten (Clean Architecture)

| Schicht | Verantwortung | Beispiele |
|---------|--------------|-----------|
| **Domain** | Geschäftslogik, Entities | `Widget`, `Config`, `Collector` Interface |
| **Application** | Use Cases, Orchestrierung | `WidgetManager`, `Scheduler`, `Orchestrator` |
| **Infrastructure** | Externe Systeme | Collectors, Config Loader, Renderer |
| **Interfaces** | API, CLI | HTTP Server, WebSocket |

**Dependency Rule**: Dependencies zeigen NUR nach innen zur Domain.

## Dokumentation

- [**SPEC.md**](SPEC.md) - Vollständige Spezifikation
- [**docs/decisions/**](docs/decisions/) - Architektur-Entscheidungen (ADRs)
- [**docs/architecture/**](docs/architecture/) - Architektur-Diagramme und Flows

## Development

### Prinzipien

1. **TDD First**: Test schreiben → Code implementieren → Refactor
2. **Commit-Logbuch**: Jeder Commit erklärt Why/What/Impact
3. **Clean Architecture**: Domain hat keine externen Dependencies
4. **Inline-Docs**: Kommentare erklären WARUM, nicht WAS

### Tests ausführen

```bash
go test ./... -v
```

### Neue Komponente hinzufügen

1. Test schreiben (rot)
2. Code implementieren (grün)
3. Refactor (sauber)
4. Commit mit Logbuch-Eintrag

Siehe [docs/decisions/001-clean-architecture.md](docs/decisions/001-clean-architecture.md) für Details.

## API

### REST Endpoints

| Methode | Endpoint | Beschreibung |
|---------|----------|--------------|
| `GET` | `/health` | Health Check |
| `GET` | `/api/v1/config` | Aktuelle Konfiguration |
| `PUT` | `/api/v1/config` | Konfiguration updaten |
| `GET` | `/api/v1/widgets` | Alle Widgets |
| `POST` | `/api/v1/widgets` | Neues Widget hinzufügen |
| `GET` | `/api/v1/system` | Live System-Infos |

### WebSocket

`ws://localhost:8080/api/v1/ws` - Echtzeit-Updates bei Widget-Änderungen

## Config-Beispiel

```yaml
background:
  type: gradient
  colors: ["#1a1a2e", "#16213e"]

widgets:
  - type: clock
    position: { x: 20, y: 20 }
    size: { width: 200, height: 50 }
    interval: 1s
    style:
      font: "Monospace 12"
      color: "#ffffff"

  - type: cpu
    position: { x: 20, y: 80 }
    size: { width: 200, height: 50 }
    interval: 2s

api:
  enabled: true
  port: 8080
  websocket: true
```

## Projekt-Status

| Komponente | Status |
|------------|--------|
| Domain Layer | ✅ Fertig |
| Application Layer | ✅ Fertig |
| Collectors (7) | ✅ Fertig |
| Config Loader | ✅ Fertig |
| Image Renderer | ✅ Fertig |
| API Server | ✅ Fertig |
| WebSocket | ✅ Fertig |
| Wayland Renderer | ⏳ Future |

## Lizenz

Siehe [LICENSE](LICENSE)
