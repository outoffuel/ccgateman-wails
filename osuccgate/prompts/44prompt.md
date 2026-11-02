This job failed because the Windows build step was using the wrong shell syntax.

The workflow file at .github/workflows/release.yml defines this step:

- Build application
- runs in the Windows matrix job
- script contains a bash-style conditional:
  if [ "${{ runner.os }}" = "Linux" ]; then
  ...
  else
  ...
  fi

On GitHub Actions, the default shell for Windows runners is PowerShell, not bash. PowerShell expects:

if ("value" -eq "Linux") { ... } else { ... }

But the script used bash syntax, so PowerShell threw this error:

- Missing '(' after 'if' in if statement.

The log confirms it:

- Line 2 | if [ "Windows" = "Linux" ]; then
- Missing '(' after 'if' in if statement.

So the failure is not from Go or Wails itself; it’s from a shell mismatch in the Windows job. The script was written as if it were running under bash, but it was being executed under PowerShell.
でwindows向けビルドがアボートされました

