# claude-foreman — Plan d'implémentation

## Vue d'ensemble

TUI (terminal) en Go avec Bubble Tea qui affiche en temps réel l'arbre des sessions tmux, leurs fenêtres, panes, processus en cours et statut des sessions Claude Code. Tourne dans une fenêtre terminal séparée, indépendante de tmux. Permet de naviguer et switcher de fenêtre tmux via clavier/souris.

La détection du statut Claude se fait par **scraping du contenu des panes tmux** (`tmux capture-pane`) et analyse des dernières lignes via regex — zéro dépendance externe, pas de hooks à installer.

---

## Architecture

```
cmd/
  main.go                  # Point d'entrée, initialisation du programme Bubble Tea

internal/
  model/
    model.go               # Elm model : état global de l'app (arbre tmux, curseur, etc.)
    update.go              # Elm update : gestion des messages (tick, input, mouse)
    view.go                # Elm view : rendu pur Model → string
    messages.go            # Définition de tous les messages (TmuxStateMsg)

  domain/
    types.go               # Types du domaine : Session, Window, Pane, Process, ClaudeStatus

  tmux/
    client.go              # Interface Client + RealClient (exécute tmux, retourne lignes brutes)
    parser.go              # Fonctions pures : parsing des sorties tmux → types domaine

  claude/
    status.go              # Analyzer : analyse le contenu capturé d'un pane pour déduire le statut Claude

  process/
    inspector.go           # Interface Inspector + implémentation via /proc

  polling/
    poller.go              # Orchestration du polling : assemble tmux + process + claude → état complet

  style/
    theme.go               # Styles Lip Gloss : couleurs, bordures, indicateurs

flake.nix                  # Nix flake : build + dev shell
```

---

## Domaine (types)

```
ClaudeStatus = idle | busy | waiting | done | none

Process {
  PID       int
  Command   string
}

ClaudeSession {
  Status    ClaudeStatus
}

Pane {
  Index          int
  PID            int
  Active         bool
  CurrentCommand string
  Processes      []Process
  Claude         *ClaudeSession    // nil si pas de claude dans ce pane
}

Window {
  Index   int
  Name    string
  Active  bool
  Panes   []Pane
}

Session {
  Name     string
  Attached bool
  Active   bool
  Windows  []Window
}

AppState {
  Sessions       []Session
  ActiveTarget   string        // "session:window:pane"
}
```

---

## Interfaces (ports)

### TmuxClient
- `ListSessions() → []string, error`
- `ListWindows() → []string, error`
- `ListPanes() → []string, error`
- `ActiveTarget() → string, error`
- `SwitchClient(target string) → error`
- `CapturePane(target string) → string, error`

L'implémentation réelle exécute les commandes tmux avec des format strings fixes et retourne les lignes brutes. Le parsing est fait dans des fonctions pures séparées.

### ProcessInspector
- `Children(pid int) → []Process, error`

Parcourt `/proc/<pid>/task/<tid>/children` récursivement pour construire l'arbre de processus d'un pane.

### Analyzer (claude)
- `Analyze(paneContent string) → ClaudeStatus`

Analyse les ~10 dernières lignes non-vides du contenu capturé d'un pane tmux. Détection par regex :
- **waiting** : patterns de prompts de permission (`(y = yes`, `Allow`, `Approve`, etc.)
- **busy** : spinner unicode + verbe en `-ing` (indicateur d'outil en cours)
- **idle** : défaut si aucun pattern ne matche

---

## Polling & assemblage

Un **poller** unique orchestre le cycle :

1. Appelle `TmuxClient` pour récupérer sessions, windows, panes (lignes brutes)
2. Parse les lignes via les fonctions pures du package `tmux`
3. Pour chaque pane, appelle `ProcessInspector.Children` pour lister les processus enfants
4. Pour chaque pane identifié comme Claude (via `isClaudePane` — cherche "claude" dans la commande courante ou les enfants) :
   - Capture le contenu du pane via `TmuxClient.CapturePane`
   - Analyse le contenu via `Analyzer.Analyze`
5. Assemble le tout en un `AppState` complet
6. Envoie un `TmuxStateMsg` au model Bubble Tea

L'intervalle de polling est de **500ms**. Le poller tourne via un cycle `tickMsg` → `pollCmd` → `TmuxStateMsg` → `tickCmd`.

---

## Model Bubble Tea

### State
- `AppState` — l'arbre complet
- `Cursor` — index dans la liste plate des éléments navigables
- `Items` — liste plate `[]NavItem` (session ou window), recalculée à chaque nouveau state

### Update
- `TmuxStateMsg` → met à jour `AppState`, recalcule `Items`, ajuste `Cursor`, lance `tickCmd`
- `tickMsg` → lance `pollCmd`
- `tea.KeyMsg` (↑/↓/j/k) → déplace `Cursor`
- `tea.KeyMsg` (Enter) → appelle `SwitchClient` vers la cible du curseur
- `tea.MouseMsg` (click gauche) → identifie la ligne cliquée, met à jour `Cursor`, switch si c'est une window
- `tea.KeyMsg` (q/Ctrl+C) → quitte

### View (fonction pure)
Itère sur `AppState.Sessions` et rend l'arbre avec Lip Gloss :
- **Session** : nom, badge "attached" si attachée, highlight si active
- **Window** : index:nom, highlight si active, statut Claude inline si présent
  - Statut Claude inline : `● busy`, `● waiting`, `● idle`, `● done`
  - Multi-Claude : `2× ● busy` si tous pareil, `(● busy · ● waiting)` si mixte
- **Footer** : résumé (nombre sessions, nombre Claude par état)

---

## Détection Claude — Pane Scraping

Approche zéro-config : pas de hooks, pas de fichiers de statut, pas de configuration utilisateur.

1. **Identification** : `isClaudePane` — un pane est Claude si sa commande courante ou un de ses processus enfants contient "claude"
2. **Capture** : `tmux capture-pane -t <target> -p -J` — récupère le contenu visible du pane
3. **Analyse** : regex sur les ~10 dernières lignes non-vides :
   - `waiting` : `(?i)\(y\s*=\s*yes|Allow|Approve|Do you want|yes.*to proceed`
   - `busy` : spinner unicode + `\w+ing\b`
   - `idle` : défaut

---

## Nix Flake

Le `flake.nix` expose :

- `packages.default` — le binaire `claude-foreman` construit via `buildGoModule` (vendor)
- `devShells.default` — shell de dev avec Go, gopls, gotools, tmux, jq

Pas de hook à installer, pas de configuration à patcher. L'installation se résume à :

```
nix profile install github:LeVraiBaptiste/claude-foreman
```

Ou dans une config NixOS/Home Manager :

```nix
environment.systemPackages = [ inputs.claude-foreman.packages.${system}.default ];
```

---

## Règles de code

1. **Fonctionnel et déclaratif** — les transformations de données sont des fonctions pures. Pas de mutation d'état en dehors du cycle Elm (Update).
2. **Interfaces pour les I/O** — tmux et process. L'implémentation réelle est injectée au main. Tout est stubable.
3. **Parsing séparé de l'exécution** — les clients retournent des données brutes (string), les parsers sont des fonctions pures testables.
4. **Zéro logique métier dans View** — le View ne fait que du rendu. Toute dérivation est pré-calculée dans Update.
5. **Un seul goroutine pour le polling** — pas de concurrence complexe. Le tick Bubble Tea déclenche un Cmd qui fait le polling sync et renvoie un message.
6. **Fail silently sur les erreurs de polling** — si tmux ou /proc est indisponible, on garde le dernier état connu. Pas de crash.
7. **Pas d'abstraction prématurée** — des structs, des interfaces, des fonctions.

---

## État d'implémentation

- [x] Scaffold — flake.nix, go.mod, main.go avec Bubble Tea
- [x] Domaine — types dans domain/types.go
- [x] Tmux client + parser — list sessions/windows/panes, capture-pane, parsing
- [x] Process inspector — lecture de /proc pour les enfants d'un PID
- [x] Claude analyzer — détection du statut par scraping pane
- [x] Poller + assemblage — orchestration + assemblage des données
- [x] Model/Update/View — TUI complet avec navigation et rendu de l'arbre
- [x] Switch client — action sur Enter/click pour changer de fenêtre tmux
- [x] Flake complet — packaging Nix
