# Recipes

Step-by-step procedures for the changes this codebase sees most often. Each one ends with
how to check the result. They are written for coding agents and work as checklists for people.

| Recipe | Use it when |
|---|---|
| [add-module.md](add-module.md) | A new resource needs its own table and endpoints |
| [add-endpoint.md](add-endpoint.md) | An existing module needs one more route |
| [add-migration.md](add-migration.md) | The schema or a query changes |
| [add-scheduled-job.md](add-scheduled-job.md) | Something has to run on a timer |

`internal/example` is the reference module: when a recipe and the code disagree, the code wins,
and the recipe should be fixed in the same change.

Rules that apply to every recipe:

- No narrating comments. A comment says why, not what.
- Finish with `make test`; fix what fails before calling the work done.
- Do not start the API (`make run`, `make dev`). The owner runs it.
- After any change to request or response types run `make swagger`, then `make api-types` in gogo-front.
