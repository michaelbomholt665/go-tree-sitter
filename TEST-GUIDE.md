# Windows VM test guide: one Python grammar

Use a Windows 11 x64 VM and PowerShell. Run **one command at a time**. This guide builds only Python for Windows amd64. Do not run `go test ./...` or select all grammars in the wizard.

`install-windows-tools.ps1` installs the toolchains, fetches Go dependencies, and creates the test configuration. Separate installation commands are not required.

WARNING: Use a disposable source copy. The script installs Windows tools and updates the User PATH. Windows can request administrator approval.

## 1. Get the script and source

Get the updated `install-windows-tools.ps1` from the project owner. Choose **one** source route below.

### Route A: Use the owner's source archive (recommended)

This route includes uncommitted fixes that a Git clone cannot retrieve.

1. Extract the source archive into a new folder.
2. Open PowerShell in that folder, where `go.mod` and `install-windows-tools.ps1` exist.
3. Start recording the session.

   ```powershell
   Start-Transcript -Path .\windows-vm-setup.log
   ```

4. Install the tools and prepare this source copy.

   ```powershell
   powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\install-windows-tools.ps1 install -RepoDirectory .
   ```

   Expect numbered stages for tool installation, verification, dependency downloads, and Windows configuration. If a stage fails, stop and send its error to the owner.

5. Stop recording setup.

   ```powershell
   Stop-Transcript
   ```

Continue with section 2. Do not run Route B.

### Route B: Let the script clone the repository

Use this route only if the owner confirms that the public default branch contains the fixes. Otherwise, request the source archive or the project's test commit.

1. Open PowerShell in the folder containing the updated installer script.
2. Start recording setup.

   ```powershell
   Start-Transcript -Path .\windows-vm-setup.log
   ```

3. Install the tools, clone the repository, and download dependencies.

   ```powershell
   powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\install-windows-tools.ps1 install
   ```

   The default checkout is `$HOME\source\go-tree-sitter-windows-test`. To select a project commit, add `-Revision` followed by the commit supplied by the owner. This parameter applies only to a new clone. The script never switches an existing working tree.

4. Stop recording setup.

   ```powershell
   Stop-Transcript
   ```

5. Enter the new checkout.

   ```powershell
   Set-Location "$HOME\source\go-tree-sitter-windows-test"
   ```

## 2. Prepare the test session

Close PowerShell after setup. Open a new PowerShell window **in the prepared checkout**. This loads the updated User PATH.

The installer creates:

- `windows-vm-config.yaml`: JSON-formatted YAML containing only Python and the installed GCC version.
- `.env`: `TS_BUILD_ENV=DEVELOPMENT`, if no `.env` already exists. The script preserves existing `.env` files.

The repository's `tree-sitter-config.yaml` stays unchanged. Every test command selects the isolated configuration explicitly. This also avoids the separate user configuration under `%AppData%\ts-build`.

1. Confirm the checkout and generated configuration.

   ```powershell
   Get-Item .\go.mod, .\windows-vm-config.yaml
   ```

2. Start recording the test.

   ```powershell
   Start-Transcript -Path .\windows-vm-test.log
   ```

3. Load the generated configuration.

   ```powershell
   $config = Get-Content .\windows-vm-config.yaml -Raw | ConvertFrom-Json
   ```

4. Select its installed C compiler for this PowerShell session.

   ```powershell
   $env:CC = '"' + $config.targets.windows.compiler + '"'
   ```

5. Select its C++ compiler.

   ```powershell
   $env:CXX = '"' + $config.targets.windows.cxx + '"'
   ```

6. Enable Go's C bindings.

   ```powershell
   $env:CGO_ENABLED = '1'
   ```

7. Confirm Go is available. Expect Go 1.26.0 or newer and `windows/amd64`.

   ```powershell
   go version
   ```

8. Refuse to reuse an earlier test's catalog.

   ```powershell
   if (Test-Path .\windows-vm-output) { throw 'Use a fresh test checkout. Do not overwrite an existing catalog.' }
   ```

If a command fails, stop. Run `$LASTEXITCODE` separately after a failing native command to record its exit code. Preserve the error and transcript.

## 3. Build and publish Python only

All commands in this section must succeed. `go run` compiles the app and uses its downloaded Go dependencies. It does not build all grammar libraries.

1. Generate the pinned Python grammar sources.

   ```powershell
   go run ./cmd/ts-build --config ./windows-vm-config.yaml --verbose build --language python
   ```

2. Compile the Python DLL for Windows amd64.

   ```powershell
   go run ./cmd/ts-build --config ./windows-vm-config.yaml --verbose compile --language python --os windows --arch amd64
   ```

3. Inspect the DLL and its provenance sidecar.

   ```powershell
   Get-ChildItem .\windows-vm-build\python\amd64\*.dll*
   ```

4. Validate and publish Python. Keep the workspace for the safety checks.

   ```powershell
   go run ./cmd/ts-build --config ./windows-vm-config.yaml --verbose move --language python --compact --clean=false
   ```

   Expect native DLL validation and a new `windows-vm-output\python` directory. Do not add `--allow-cross-validation`: this test requires native Windows validation.

5. List the published files.

   ```powershell
   Get-ChildItem .\windows-vm-output\python -Recurse
   ```

   Expect the DLL, `manifest.json`, `node-types.json`, and `compact-node-types.yaml`. Query files appear if upstream supplies them.

6. Inspect the manifest. Check that it identifies Windows amd64 and parser ABI 15.

   ```powershell
   Get-Content .\windows-vm-output\python\manifest.json
   ```

7. Verify the compact schema without rewriting it.

   ```powershell
   go run ./cmd/ts-build --config ./windows-vm-config.yaml compact --language python --check
   ```

The installer also supports `-RunGrammarTest` for automated execution of this pipeline. Do not use that switch before this manual test: it already publishes the catalog.

## 4. Check safe failures

The three commands identified below intentionally return nonzero exit codes. Every other command must succeed.

### Reject an ambiguous clean command

1. Pass Python as an invalid positional argument. Expect an argument error, not cleanup.

   ```powershell
   go run ./cmd/ts-build --config ./windows-vm-config.yaml clean python
   ```

2. Confirm that the Python sources still exist.

   ```powershell
   Get-Item .\windows-vm-build\python\amd64\src\parser.c -ErrorAction Stop
   ```

### Preserve an existing catalog

Windows supports first publication into an absent catalog. Atomic replacement of an existing catalog is currently unsupported. Both attempts below must leave the published files unchanged.

1. Record the files and their checksums.

   ```powershell
   $before = Get-ChildItem .\windows-vm-output -Recurse -File | Sort-Object FullName | Get-FileHash -Algorithm SHA256 | Select-Object Path, Hash | ConvertTo-Json -Compress
   ```

2. Attempt publication without `--force`. Expect an error requiring `--force`.

   ```powershell
   go run ./cmd/ts-build --config ./windows-vm-config.yaml --verbose move --language python --compact --clean=false
   ```

3. Attempt forced replacement. Expect an error explaining that atomic directory exchange is unsupported.

   ```powershell
   go run ./cmd/ts-build --config ./windows-vm-config.yaml --verbose move --language python --compact --clean=false --force
   ```

4. Record the files and checksums again.

   ```powershell
   $after = Get-ChildItem .\windows-vm-output -Recurse -File | Sort-Object FullName | Get-FileHash -Algorithm SHA256 | Select-Object Path, Hash | ConvertTo-Json -Compress
   ```

5. Verify that the catalog is unchanged.

   ```powershell
   if ($before -ne $after) { throw 'FAIL: the published catalog changed.' } else { Write-Output 'PASS: the published catalog is unchanged.' }
   ```

## 5. Report the result

1. Stop recording.

   ```powershell
   Stop-Transcript
   ```

Send the project owner:

- `windows-vm-setup.log` from the folder where setup started.
- `windows-vm-test.log`.
- `windows-vm-config.yaml`.
- `windows-vm-output\python\manifest.json`, if publication succeeded.
- The source archive or project commit used.
- The first unexpected failing command and its exit code, if a step failed.

If setup appears stuck, report the last visible stage. The installer displays its WinGet checks instead of hiding prompts. Installer elevation, downloads, or antivirus scanning can still take time. Run `winget --logs` to locate WinGet diagnostics.

A successful test publishes one validated Python DLL and preserves the catalog after both rejected replacements. It does not verify descendant cancellation, WASM, Zig wrappers, other grammars, or other architectures.

Preserve the workspace until the owner reviews the results. The script's `uninstall` action removes Go, Git, Node.js, and WinLibs, including pre-existing installations. It requests confirmation and does not delete the checkout or test output.
