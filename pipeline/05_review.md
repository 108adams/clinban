# Clinban Pipeline Review

Reviewed documents, top to bottom:

1. `pipeline/00_vision.md`
2. `pipeline/01_requirements.md`
3. `pipeline/02_architecture.md`
4. `pipeline/03_design.md`
5. `pipeline/04_tasks.md`

## Executive Findings

The plan is coherent in broad shape: markdown ticket files, a Go CLI, lint as the schema safety net, and a small internal package layout. The main risk is not missing implementation detail; it is inconsistent contracts between stages. Several downstream design and task decisions encode interpretations that differ from the vision or requirements.

Resolve these before writing code:

- Decide whether `id` is a YAML integer or a quoted string. The requirements say integer, while architecture/design/tasks use string.
- Decide whether closing a ticket automatically archives it, or whether `clinban archive` is a separate explicit operation.
- Decide whether tickets are identified by schema or by filename. The documents currently claim schema identity while requiring filename-derived lookup and lint checks.
- Decide whether `clinban show` is view-only/rendering, or editor-based mutation.
- Fix the non-interactive write flow so invalid automata-created tickets are not written unless that is intentionally part of the product contract.
- Define parse-error behavior separately from lint. Lint cannot validate malformed frontmatter after parsing fails.

## 00 Vision Review

### Concern: automatic archive is promised, but later stages make archive explicit

`00_vision.md` says closed tickets move to `archive/` automatically and repeats that as v1 acceptance: closing a ticket moves its file to `archive/` automatically. Later requirements add `clinban archive`, and `clinban move <id> done` only updates status. See `00_vision.md:38`, `00_vision.md:72`, `01_requirements.md:217`, and `01_requirements.md:226`.

This is a product-level contradiction. If `done` means closed, then `move ... done` should archive immediately. If `done` means completed-but-still-active, then the vision acceptance criterion should be rewritten.

Recommendation: choose one model:

- Automatic model: `clinban move <id> done` updates status and moves the file to archive.
- Manual model: `done` remains active until `clinban archive` is run; update the vision to say archiving is explicit.

### Concern: `show` is described as rendering but later becomes editing

The vision acceptance says `clinban show <id>` renders a ticket readably in the terminal. Requirements and tasks define `show` as opening the ticket in `$EDITOR` and writing `updated`. See `00_vision.md:70`, `01_requirements.md:195`, and `04_tasks.md:334`.

This changes both user expectation and safety profile. A command named `show` normally should not mutate files.

Recommendation: split the behavior:

- `clinban show <id>` prints a readable view to stdout.
- `clinban edit <id>` opens `$EDITOR` and updates `updated`.

If only one command is desired, rename or explicitly define `show` as edit-mode in the vision.

### Concern: "identified by schema, not filename" conflicts with later filename rules

The vision says tickets are identified by schema, not filename pattern. Requirements and design later require filename prefixes for ID scanning, `FindByID`, and lint's ID-vs-filename rule. See `00_vision.md:38`, `01_requirements.md:46`, `01_requirements.md:262`, `03_design.md:209`, and `04_tasks.md:163`.

This matters because automata are first-class writers. If an agent writes a valid schema into `my-ticket.md`, is it a ticket? The vision says yes; lint/store behavior says no or invalid.

Recommendation: state the real invariant. A practical v1 invariant could be: "A ticket is a markdown file with valid Clinban frontmatter, and managed tickets must use `<id>-<slug>.md`; lint reports filename/schema mismatches."

### Concern: cross-system linking is first-class but disappears from v1 execution

The vision says cross-linking between knowledge base, ADRs, tickets, and code is first-class. It is only should-have in the capabilities table and does not appear in requirements, architecture, design, or tasks. See `00_vision.md:46`, `00_vision.md:47`, and `00_vision.md:99`.

Recommendation: either explicitly defer cross-linking from v1, or add a small v1 convention such as accepting raw markdown/wiki links without validation.

## 01 Requirements Review

### Concern: `id` cannot be both an integer and zero-padded

The requirements define `id` as an integer with a 4-digit zero-padded constraint. Numeric YAML values do not preserve leading zeroes semantically, and some parsers treat leading-zero numbers awkwardly. Architecture and design correctly move toward `id: "0042"` as a string. See `01_requirements.md:36`, `02_architecture.md:44`, and `03_design.md:76`.

Recommendation: make `id` a string everywhere, constrained by regex `^[0-9]{4}$` for v1. If more than 9999 tickets are possible, define the overflow behavior now.

### Concern: non-interactive creation writes before handling lint errors

`clinban new --no-interactive` runs lint, writes the file, then exits 1 if lint errors exist. That means automata can create invalid repo files through the CLI even though the automata path is supposed to be lint-protected. See `01_requirements.md:142` to `01_requirements.md:149`.

This also conflicts with the later business rule that says lint errors block only `clinban register`, not necessarily non-interactive creation. See `01_requirements.md:288`.

Recommendation: for `new --no-interactive`, validate all inputs and run lint before writing. If lint fails, do not write. Interactive creation can remain permissive because the user is in a repair loop.

### Concern: malformed YAML cannot be "caught by lint" if parsing fails first

`clinban register` says invalid YAML frontmatter is caught by lint. But lint operates on a parsed `Ticket` in design. A parse failure must be reported before lint can run. See `01_requirements.md:167`, `03_design.md:163`, and `04_tasks.md:322`.

Recommendation: define two validation phases:

- Parse/frontmatter errors: file is unreadable as a ticket.
- Lint errors: file parsed but violates schema/business rules.

Both can produce the same one-line output format, but they should not be modeled as the same engine unless lint accepts raw bytes.

### Concern: active `done` tickets are underspecified in list/sort behavior

Requirements add a manual archive command, so `done` tickets can exist in the active directory. But `clinban list` sorts only `in-progress`, `blocked`, and `backlog`; it omits `done`. See `01_requirements.md:176` and `01_requirements.md:226`.

Recommendation: add `done` to the list sort order or explicitly exclude `done` from active list output. If `done` remains active until archived, it should probably sort last.

### Concern: `show` updates `updated` before lint, which can damage invalid edits

The requirements say `show` opens the file, then updates `updated`, runs lint, and prompts on errors. If the user leaves malformed frontmatter, the tool may be unable to parse the file to update `updated`. If the user leaves schema-invalid but parseable content, writing a refreshed timestamp may normalize or rewrite content before the user accepts repairs. See `01_requirements.md:195` to `01_requirements.md:200`.

Recommendation: after editor close, parse first. If parsing fails, report the parse error and prompt to reopen without rewriting. If parsing succeeds, update `updated`, write atomically, then lint. Or better, lint first and only update/write when the edited ticket is valid.

### Concern: config discovery is incomplete at requirements level

Requirements define `.clinban` but not how the project root is found, how relative paths are resolved, or what happens when commands run from subdirectories. Design later fills this in. See `01_requirements.md:67` to `01_requirements.md:77` and `03_design.md:278`.

Recommendation: promote root discovery and relative path resolution into requirements, because it affects every command and test.

### Concern: file/schema identity remains contradictory

Business rule 8 says ticket files are identified by schema, not filename pattern, but lint requires ID to match the filename and store operations resolve by filename prefix. See `01_requirements.md:262`, `01_requirements.md:286`, and `04_tasks.md:164`.

Recommendation: update the business rule to separate "schema is the authoritative content contract" from "filename is the v1 addressing/indexing convention."

## 02 Architecture Review

### Concern: schema type changes from requirements without an explicit decision

Architecture uses a quoted string ID while requirements say integer. The architecture version is better, but it needs to be recorded as a correction or ADR because it changes the external automata contract. See `01_requirements.md:36` and `02_architecture.md:44`.

Recommendation: add an ADR or update requirements so there is one authoritative schema.

### Concern: lint has "no filesystem dependency" but uniqueness is repository-level

The lint component is defined as no filesystem dependency, which is good for unit testing. But ID uniqueness is a repo-wide rule requiring all ticket IDs from the store. See `02_architecture.md:16`, `01_requirements.md:265`, and `03_design.md:165`.

This is workable only if the architecture names the boundary clearly: store gathers repository context, lint evaluates the supplied context.

Recommendation: document `Lint(ticket, filename, repoContext)` as the contract, not pure single-ticket validation. Include path/filename and known IDs as caller-supplied inputs.

### Concern: NFR says path traversal prevention is required, but no design detail follows

Architecture marks path traversal prevention as required. Requirements accept arbitrary `register <path>`, configuration paths, and IDs from CLI args. Tasks do not add tests for path containment or symlink behavior. See `02_architecture.md:71`, `01_requirements.md:153`, and `04_tasks.md:314`.

Recommendation: define the actual security rules:

- IDs must match `^[0-9]{4}$`.
- Generated filenames are basename-only.
- `TicketPath`, archive moves, and active moves must ensure the final path stays under configured directories.
- Decide whether symlinks inside ticket directories are followed, rejected, or treated as plain files.

### Concern: POSIX-specific atomicity is assumed while targets include macOS/Linux only

Same-directory rename is the right approach for Linux/macOS, but the ADR overstates "no partial writes are ever visible" without mentioning fsync. A crash can still lose a just-written file or directory entry without syncing, depending on filesystem semantics. See `02_architecture.md:153` to `02_architecture.md:162`.

Recommendation: either scope the guarantee to "readers never observe a partially written final file during normal operation" or include file/directory fsync if crash durability is required.

### Concern: open questions include choices already locked by tasks

Architecture lists YAML, TOML, Cobra version, template approach, terminal width, and test strategy as open. Tasks later choose `yaml.v3`, `BurntSushi/toml`, embedded templates, `golang.org/x/term`, and integration tests. See `02_architecture.md:170` to `02_architecture.md:176` and `04_tasks.md:20` to `04_tasks.md:21`.

Recommendation: update architecture after task planning so open questions do not remain stale.

## 03 Design Review

### Concern: `Marshal(Parse(b)) == b` is too strong for YAML

The design requires byte-for-byte round-trip equality for any valid ticket. YAML libraries generally do not preserve quoting, key order, comments, blank lines, or timestamp formatting. See `03_design.md:94` to `03_design.md:96`.

This requirement will either fail in implementation or force a much more complex frontmatter-preserving parser.

Recommendation: replace it with semantic round-trip equality: parse/marshal/parse yields equivalent field values and body. If preserving comments/order matters, explicitly design for raw frontmatter preservation.

### Concern: `tags,omitempty` conflicts with always serializing `tags: []`

The struct tag says `yaml:"tags,omitempty"`, but constraints require empty tags to serialize as `tags: []`. See `03_design.md:80` and `03_design.md:96`.

Recommendation: remove `omitempty` or implement custom marshal behavior.

### Concern: lint rule for timestamp parsing conflicts with `time.Time` fields

The design uses `time.Time` for `Created` and `Updated`, then says lint validates RFC3339 parsing. If YAML decoding already converted into `time.Time`, invalid raw strings may fail during parse before lint, and formatting details are lost. See `03_design.md:81`, `03_design.md:82`, and `03_design.md:178`.

Recommendation: either keep timestamps as strings in the schema model and parse in lint, or treat invalid timestamps as parse errors instead of lint errors.

### Concern: `nil` tag entries cannot exist in `[]string`

The lint rule says YAML decode enforces tags as a list of strings and lint verifies no nil entries. A `[]string` cannot contain nil entries. Non-string list items will fail or coerce depending on decoder behavior before lint sees them. See `03_design.md:80` and `03_design.md:179`.

Recommendation: if lint must report `tags` type errors, parse frontmatter into a raw `map[string]any` first or use a custom schema decoder. Otherwise define these as parse/decode errors.

### Concern: `Store` loses filenames after parsing

`ListActive` returns only `[]*ticket.Ticket`, but list sorting, lint output, archive listing, and duplicate diagnostics may need filenames/paths. See `03_design.md:219` to `03_design.md:224` and `04_tasks.md:302` to `04_tasks.md:305`.

Recommendation: introduce a `Record`/`Entry` type with `Ticket`, `Path`, `Filename`, and `InArchive`, or return path alongside ticket for store list APIs.

### Concern: `WriteTicket` always mutates `Updated`, but creation and registration need controlled timestamps

`WriteTicket` always sets `Updated = time.Now()`. Creation paths set `Created = Updated = now` before writing, so `WriteTicket` can make them diverge. Register also overwrites both with the same timestamp before write. See `03_design.md:226` to `03_design.md:229`, `01_requirements.md:50` to `01_requirements.md:52`, and `04_tasks.md:238`.

Recommendation: split write concerns:

- `WriteTicket` writes exactly the given ticket.
- Command/service layer decides when to touch `Updated`.
- Or add `WriteTicket(..., TouchUpdated bool)` if the team wants store-owned timestamp mutation.

### Concern: `MoveToArchive`/`MoveToActive` do not update status

The store move APIs move files only. That is fine if command logic owns status, but the `done -> backlog` special case in tasks moves the file before updating status. Failure between those steps can leave a `done` ticket active. See `03_design.md:234` to `03_design.md:241` and `04_tasks.md:281` to `04_tasks.md:283`.

Recommendation: command flow should update status and file location as one higher-level operation, with clearly ordered failure behavior. At minimum, write updated status to the final path rather than rename then mutate.

## 04 Tasks Review

### Concern: dependency graph is missing direct task dependencies

T-17 uses `internal/template`, but the dependency graph has no task for template implementation except inside T-17. T-10 says lint all tickets in both dirs, but `Store` has no `ListAll`/archive listing API in T-08. See `04_tasks.md:360`, `04_tasks.md:216`, and `03_design.md:219`.

Recommendation: add explicit store APIs/tasks for listing archived/all tickets, and either create a separate template task or make T-17's ownership of `internal/template` explicit in the dependency graph.

### Concern: T-02 repeats the impossible byte-for-byte YAML round trip

T-02 makes byte equality a done condition. This is likely to fail with `yaml.v3` unless fixtures are constrained to the encoder's exact output. See `04_tasks.md:37` to `04_tasks.md:45`.

Recommendation: change the test to semantic equality plus body preservation, or decide to preserve raw frontmatter formatting and add that complexity intentionally.

### Concern: T-03 self-transition behavior conflicts with command behavior unless handled outside FSM

T-03 says self-transitions return an error. Requirements and T-13 say `clinban move <id> <same-status>` exits silently. This is acceptable only if T-13 checks equality before calling FSM. See `04_tasks.md:64` to `04_tasks.md:67` and `04_tasks.md:277` to `04_tasks.md:280`.

Recommendation: keep the command pre-check explicit in tests. Add a CLI test for same-status no-op so the FSM behavior does not leak to users.

### Concern: T-08 scans filenames, not schema IDs

T-08 says `NextID`, `FindByID`, and `AllIDs` use filename prefixes. This fails the "schema is authoritative" premise and can miss duplicate schema IDs if filenames differ. See `04_tasks.md:163` to `04_tasks.md:169`.

Recommendation: choose the invariant. If filename prefix is authoritative for indexing, update earlier docs. If schema is authoritative, T-08 must parse ticket frontmatter when scanning IDs.

### Concern: T-10 cannot lint all tickets with the designed store API

T-10 requires iterating all tickets in active and archive, but the store design only exposes `ListActive`, `FindByID`, and `AllIDs`. See `04_tasks.md:216` and `03_design.md:219`.

Recommendation: add `ListAll() []Record` or `ListArchive()` to `Store`.

### Concern: T-11 can fail uniqueness lint because the candidate ID is not in `allIDs`

T-11 runs lint before writing. Depending on how `allIDs` is passed, the uniqueness rule may not see the new ticket, or it may be impossible to distinguish "self" from duplicate. See `04_tasks.md:238` to `04_tasks.md:240` and `03_design.md:163` to `03_design.md:167`.

Recommendation: define uniqueness lint input as repository records plus current file identity, or make uniqueness a store-level preflight for create/register flows.

### Concern: T-15 says `MoveToActive` can register an arbitrary external path

Design defines `MoveToActive` as moving a file from `ArchiveDir` into `TicketsDir`. T-15 uses it for arbitrary external file adoption. See `03_design.md:239` to `03_design.md:241` and `04_tasks.md:326`.

Recommendation: add a separate `Adopt(path, ticket)` or `MoveIntoActive(path)` API with path validation and collision behavior.

### Concern: T-16 can rewrite invalid edits before reporting lint

T-16 reloads the edited file, updates `Updated`, writes the ticket, then runs lint. That means schema-invalid but parseable content is rewritten before errors are shown. See `04_tasks.md:340` to `04_tasks.md:345`.

Recommendation: parse and lint first, then write the timestamp update only when valid. If preserving the permissive interactive behavior is important, keep the user's raw file untouched until they accept a successful repair.

### Concern: T-17 uses cross-directory `os.Rename` from system temp to repo

T-17 writes the interactive template to system temp, then moves it to `TicketsDir` with `os.Rename`. This can fail across filesystems. The architecture specifically rejects system temp for final writes because cross-device rename is not reliable. See `04_tasks.md:362`, `04_tasks.md:367`, and `02_architecture.md:151`.

Recommendation: after editor completion, read the temp file and use `store.WriteTicket`/same-directory atomic write to create the final file, or create the editable temp file inside `TicketsDir` under a hidden/temp name and then rename in place.

### Concern: T-17 unchanged-template detection differs from requirements

Requirements compare required fields to placeholder values for `title` and `type`. T-17 only checks `title == ""`; the template sets `type: "task"`, so an untouched template could be interpreted as changed if the user only modifies body, and placeholder semantics are inconsistent. See `01_requirements.md:113` to `01_requirements.md:115`, `03_design.md:263` to `03_design.md:264`, and `04_tasks.md:364`.

Recommendation: define a clear discard rule. For example: discard if title is empty, regardless of type/body, because title is the minimum intentional signal. Then update requirements to match.

## Suggested Pre-Implementation Edits

Before implementation, update the pipeline in this order:

1. Amend `01_requirements.md` so schema types, archive behavior, `show`/`edit`, filename identity, parse-vs-lint, and list behavior are unambiguous.
2. Amend `00_vision.md` to match the chosen archive and show semantics.
3. Amend `02_architecture.md` open questions and record the ID/string decision.
4. Amend `03_design.md` to remove impossible marshal invariants, fix `tags`, clarify timestamps, and add store record/list-all APIs.
5. Amend `04_tasks.md` so task dependencies and done criteria match the corrected design.

The project is close to implementable, but these contract mismatches should be fixed first. Otherwise the implementation will have to choose behavior ad hoc, and tests will encode accidental decisions instead of product decisions.
