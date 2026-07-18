# myenv CI demo

`pass/` should pass. `fail/` deliberately fails:

- `MISSING_SCHEMA_VAR` appears in code but not schema.
- encrypted `PORT` is not an integer.
- encrypted `API_URL` uses `http`, while schema requires `https`.
- encrypted `EXTRA_DOTENV` has no schema declaration.

Both folders contain encrypted dummy values only. They share one test key. Add
that key to repository Actions secrets as `MYENV_DECRYPT_KEY`, then run the
manual **myenv CI demo** workflow from GitHub Actions.

The workflow verifies both outcomes: pass fixture succeeds; fail fixture must
fail inside its action step.