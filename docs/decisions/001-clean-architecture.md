# ADR-001: Clean Architecture + DDD für Dynamic Background

## Status
Accepted

## Kontext
Das Projekt erzeugt einen dynamischen Sway-Wayland-Hintergrund mit Systeminformationen.
Der Entwickler implementiert nicht selbst, sondern beauftragt eine KI. Dies erzeugt
Brain-Drain-Risiko: Der Entwickler versteht am Ende sein eigenes Projekt nicht.

## Entscheidung
Wir verwenden Clean Architecture mit Domain-Driven-Design Elementen:

### Schichten (Dependency Rule: zeigt nach innen)
```
Interfaces → Application → Domain ← Infrastructure
```

- **Domain**: Entities (Widget, Config), Interfaces (Collector), Value Objects
- **Application**: Use Cases (Widget orchestrieren, Daten sammeln, rendern)
- **Infrastructure**: Wayland, Cairo, /proc, HTTP Server
- **Interfaces**: API, CLI

### Begründung

| Kriterium | Warum Clean Architecture |
|-----------|-------------------------|
| Testbarkeit | Domain hat keine externen Dependencies; unit-testbar |
| Wissenstransfer | Klare Schichten = verständliche Struktur |
| Erweiterbarkeit | Neue Widgets via Interface, keine Änderungen am Core |
| Wartbarkeit | Jede Schicht hat eine klare Responsibility |

### Alternativen verworfen

| Ansatz | Warum verworfen |
|--------|----------------|
| Monolithisch | Schnell am Anfang, aber untestbar und unverständlich |
| MVC | Zu UI-lastig; passt nicht für Background-Rendering |
| Hexagonal | Ähnlich, aber Clean Architecture ist expliziter über Schichten |

## TDD Workflow
Strict Test-Driven Development:
1. Test schreiben (rot)
2. Code implementieren (grün)
3. Refactor (sauber)
4. Commit mit Logbuch-Eintrag (Warum/Wozu/Impact)

## Commit-Logbuch Format
Jeder Commit enthält:
- **WHY**: Warum wurde der Code geschrieben?
- **WHAT**: Was tut er?
- **IMPACT**: Was passiert wenn er fehlt?

## Konsequenzen

### Positiv
- Jede Änderung ist durch Tests abgesichert
- Neue Entwickler finden sich schnell zurecht
- Domain-Logik ist isoliert und verständlich
- Infrastruktur kann ausgetauscht werden (z.B. Wayland → X11)

### Negativ
- Mehr Boilerplate am Anfang
- Langsamere initiale Entwicklung
- Mehr Dateien zu navigieren

### Risiken
- Overengineering für ein kleines Projekt
- **Mitigation**: MVP zuerst, Komplexität nur wo nötig
