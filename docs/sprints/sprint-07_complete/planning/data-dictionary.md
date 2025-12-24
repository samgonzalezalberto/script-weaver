# Project Integration &mdash; Data Dictionary

## Workspace

### .scriptweaver/

* Type: directory
* Required: yes (auto-created)
* Description: Isolated ScriptWeaver workspace

## Workspace Subdirectories

### cache/

* Type: directory
* Description: Hash-addressed artifacts

### runs/

* Type: directory
* Description: Execution run records

### logs/

* Type: directory
* Description: Deterministic logs

## Configuration File

### config.json

* Type: object
* Required: no
* Description: Integration configuration only

Allowed Fields:

* graph_path

Disallowed

* Semantic overrides
* workspace_path

## Graph Locations

### graphs/

* Type: directory
* Description: Conventional graph location

### .scriptweaver/graphs/

* Type: directory
* Description: Workspace-local graphs

## Determinism Notes

* Paths resolved deterministically
* No implicit environment lookups
* No global state
