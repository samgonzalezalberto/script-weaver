# Sprint-09 &mdash; Data Dictionary

## Top-Level

### plugin_manifest

* Type: object
* Required: yes
* Description: Declares a plugin and its capabilities

### runtime_plugin_state

* Type: object
* Required: no (runtime only)
* Description: In-memory state tracked for each loaded plugin

## Plugin Manifest Object

### plugin_id

* Type: string
* Required: yes
* Example: `"logging-plugin"`
* Description: Globally unique identifier for the plugin

### version

* Type: string
* Required: yes
* Example: `"0.1.0"`
* Description: Plugin version identifier

### hooks

* Type: array (string)
* Required: yes
* Description: Lifecycle hooks the plugin registers for

*Allowed values*:

* `BeforeRun`
* `AfterRun`
* `BeforeNode`
* `AfterNode`

### description

* Type: string
* Required: no
* Description: Human-readable description of the plugin

## Runtime Plugin State Object

### plugin_id

* Type: string
* Required: yes
* Description: References to the associated plugin manifest

### enabled

* Type: boolean
* Required: yes
* Description: Whether the plugin is active for the current run

### load_error

* Type: string
* Required: no
* Description: Error message if the plugin failed to load or register

## Plugin Directory Layout (Logical)

### plugins_root

* Type: directory
* Required: yes
* Description: Root directory for plugin discovery (e.g. `.scriptweaver/plugins/`)

### plugin_directory

* Type: directory
* Required: yes
* Description: One directory per plugin, non-recursive

### manifest_file

* Type: file
* Required: yes
* Description: File containing the plugin manifest definition (must be named `manifest.json`)

## Determinism Notes

* Plugin discovery order is deterministic
* Plugin execution order is deterministic per hook
* Absence of plugins produces no side effects
* Invalid plugins are skipped but logged

## Hash / Core State Interaction Rules

Included in core execution hash:

* None (plugins do not affect graph hashing in this sprint)

Excluded:

* plugin_manifest
* runtime_plugin_state
* plugin directory contents