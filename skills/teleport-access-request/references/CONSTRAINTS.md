# Resource Constraint Syntax

A `--resource` value passed to `tsh request create` takes one of three forms.
The form is selected by the shape of the string.

## 1. Plain resource ID (unconstrained)

A slash-delimited resource ID requests the resource with no scoping, granting
every login or role the requester is allowed to use on it:

```
/<cluster>/<kind>/<name>
```

Example: `/main/node/web-1`

## 2. Inline constraints (anchored-key grammar)

Append `|<key>=<v1>,<v2>` to a resource ID to scope the grant to a subset of
principals. Multiple keys are joined with `|`:

```
/<cluster>/<kind>/<name>|<key>=<v1>,<v2>
```

Examples:

```
/main/node/web-1|logins=root,admin
/main/app/aws-console|role_arns=arn:aws:iam::123456789012:role/ReadOnly
```

The `|` is only treated as a constraint separator when immediately followed by
a recognized key and `=`, so resource names that themselves contain `|` still
parse correctly.

## 3. JSON ResourceAccessID (for automation)

A single JSON object, the canonical machine-generated form. Useful for large or
scripted requests, and as the element type of the `--resource-file` list:

```json
{"id":{"cluster":"main","kind":"node","name":"web-1"},"constraints":{"version":"v1","ssh":{"logins":["root","admin"]}}}
```

## Recognized keys

| Key         | Applies to kind     |
|-------------|---------------------|
| `logins`    | `node`              |
| `role_arns` | `app` (AWS console) |

A constraint key must match the resource kind (for example `logins` only
attaches to a `node`), and every value must be one the requester is allowed to
use on that resource. Both are checked when the request is created. Any other
key is rejected with an error.

## If constraints are not supported

If the cluster is too old to understand constraints, creating a request that
carries them fails. Retry with the constraints removed to request broader,
unconstrained access to the resource.
