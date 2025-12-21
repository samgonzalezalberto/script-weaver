# Graph Schema v1 &mdash; Data Dictionary

## Top-Level

### schema_version

* Type: string
* Required: yes
* Example: "1.0.0"
* Description: Schema version identifier

### graph

* Type: object
* Required: yes
* Description: Execution graph definition

### metadata

* Type: object
* Required: yes
* Description: Non-execution metadata

## Graph Object

### nodes

* Type: array
* Required: yes
* Description: List of execution nodes

### edges

* Type: array
* Required: yes
* Description: Directed dependencies between nodes

## Node Object

### id

* Type: string
* Required: yes
* Description: Globally unique node identifier

### type

* Type: string
* Required: yes
* Description: Node execution type

### inputs

* Type: object (map<string, any>)
* Required: yes
* Description: Immutable input parameters

### outputs

* Type: array (string)
* Required: yes
* Description: Declared output keys

## Edge Object

### from

* Type: string
* Required: yes
* Description: Source node ID

### to

* Type: string
* Required: yes
* Description: Destination node ID

## Metadata Object

### name

* Type: string
* Required: no
* Description: Human-readable graph name

### description

* Type: string
* Required: no
* Description: Long-form description

### labels

* Type: array
* Required: no
* Description: Arbitrary tags

## Determinism Notes

* All fields are explicitly typed
* No implicit defaults unless specified
* Unknown fields are rejected

## Hash Inclusion Rules

Included:

* graph.nodes (sorted by id)
* graph.edges (sorted by from, to)

Excluded:

* metadata
* schema_version
