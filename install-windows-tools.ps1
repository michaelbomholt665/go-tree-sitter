[CmdletBinding()]
param(
    [Parameter(Position = 0)]
    [ValidateSet('install', 'uninstall', 'help')]
    [string]$Action = 'install',

    [string]$WinLibsVersion = $env:WINLIBS_VERSION,

    [string]$RepoUrl = 'https://github.com/michaelbomholt665/go-tree-sitter.git',

    [string]$RepoDirectory = (Join-Path $env:USERPROFILE 'source\go-tree-sitter-windows-test'),

    [string]$Revision = '',

    [switch]$RunGrammarTest
)

$ErrorActionPreference = 'Stop'

$StateFile = Join-Path $env:USERPROFILE '.go-tree-sitter-win-tools-paths'
$WingetPackages = @(
    'GoLang.Go',
    'Git.Git',
    'OpenJS.NodeJS.LTS',
    'BrechtSanders.WinLibs.POSIX.UCRT'
)

function Show-Usage {
    @'
Usage: powershell -ExecutionPolicy Bypass -File .\install-windows-tools.ps1 [install|uninstall|help]

Commands:
  install     Install and verify tools, clone or reuse the repository, fetch Go
              dependencies, and create an isolated Windows/Python configuration.
              Add -RunGrammarTest to build, compile, and publish only Python.
              (Default when no command is given.)
  uninstall   Uninstall tree-sitter-cli and the winget packages installed above,
              unset CGO_ENABLED, and remove PATH entries added by this script.
  help        Show this help message.

Parameters / Environment variables:
  -WinLibsVersion / $env:WINLIBS_VERSION
              Optional exact winget version for BrechtSanders.WinLibs.POSIX.UCRT.
  -RepoUrl    Git clone URL. Defaults to the project's public GitHub repository.
  -RepoDirectory
              Destination checkout. Defaults to ~/source/go-tree-sitter-windows-test.
              Use -RepoDirectory $PWD.Path to test the current source copy.
  -Revision   Optional project commit or branch to check out in a NEW clone.
              Existing working trees are never reset or switched.
  -RunGrammarTest
              Run the Python-only Windows amd64 pipeline after setup.
              No Go test suite, other grammars, or WASM are built.

Use a revision containing the fixes under test. A default-branch clone excludes
uncommitted changes. No normal grammar catalog or repository config is replaced.
Uninstall removes the listed tools, including pre-existing installations.
Uninstall does not remove the checkout, test output, or project dependencies.
'@
}

function Assert-WingetAvailable {
    if (-not (Get-Command winget -ErrorAction SilentlyContinue)) {
        throw 'winget is not available on PATH. Install App Installer / winget and try again.'
    }
}

function Get-PathParts {
    param([string]$RawPath)
    if ([string]::IsNullOrWhiteSpace($RawPath)) {
        return @()
    }
    return @(
        $RawPath -split ';' |
            ForEach-Object { $_.Trim() } |
            Where-Object { $_ -ne '' }
    )
}

function Refresh-ProcessPath {
    $machine = Get-PathParts ([Environment]::GetEnvironmentVariable('Path', 'Machine'))
    $user = Get-PathParts ([Environment]::GetEnvironmentVariable('Path', 'User'))

    $seen = New-Object 'System.Collections.Generic.HashSet[string]' ([System.StringComparer]::OrdinalIgnoreCase)
    $combined = New-Object 'System.Collections.Generic.List[string]'

    foreach ($entry in ($machine + $user)) {
        $normalized = $entry.TrimEnd('\')
        if ($seen.Add($normalized)) {
            $combined.Add($entry)
        }
    }

    $env:Path = ($combined -join ';')
}

function Find-ToolDirectory {
    param(
        [string]$ExeName,
        [string]$Kind
    )

    $candidates = New-Object 'System.Collections.Generic.List[string]'

    switch ($Kind) {
        'go' {
            if ($env:ProgramFiles) { $candidates.Add((Join-Path $env:ProgramFiles 'Go\bin')) }
            if (${env:ProgramFiles(x86)}) { $candidates.Add((Join-Path ${env:ProgramFiles(x86)} 'Go\bin')) }
            $candidates.Add('C:\Go\bin')
        }
        'git' {
            if ($env:ProgramFiles) { $candidates.Add((Join-Path $env:ProgramFiles 'Git\cmd')) }
            if (${env:ProgramFiles(x86)}) { $candidates.Add((Join-Path ${env:ProgramFiles(x86)} 'Git\cmd')) }
        }
        'node' {
            if ($env:ProgramFiles) { $candidates.Add((Join-Path $env:ProgramFiles 'nodejs')) }
            if (${env:ProgramFiles(x86)}) { $candidates.Add((Join-Path ${env:ProgramFiles(x86)} 'nodejs')) }
        }
        'npm-global' {
            if ($env:APPDATA) { $candidates.Add((Join-Path $env:APPDATA 'npm')) }
        }
        'winlibs' {
            $pkgRoot = Join-Path $env:LOCALAPPDATA 'Microsoft\WinGet\Packages'
            if (Test-Path $pkgRoot) {
                Get-ChildItem -Path $pkgRoot -Filter 'BrechtSanders.WinLibs*' -Directory -ErrorAction SilentlyContinue |
                    ForEach-Object {
                        Get-ChildItem -Path $_.FullName -Filter $ExeName -Recurse -ErrorAction SilentlyContinue |
                            ForEach-Object { $candidates.Add($_.DirectoryName) }
                    }
            }
            $candidates.Add((Join-Path $env:LOCALAPPDATA 'Microsoft\WinGet\Links'))
            $candidates.Add('C:\mingw64\bin')
        }
    }

    foreach ($dir in $candidates) {
        if ($dir -and (Test-Path (Join-Path $dir $ExeName))) {
            return $dir.TrimEnd('\')
        }
    }
    return $null
}

function Ensure-ToolInUserPath {
    param(
        [string]$ExeName,
        [string]$Kind
    )

    $machine = Get-PathParts ([Environment]::GetEnvironmentVariable('Path', 'Machine'))
    $user = Get-PathParts ([Environment]::GetEnvironmentVariable('Path', 'User'))
    $all = $machine + $user

    foreach ($dir in $all) {
        if (Test-Path (Join-Path $dir $ExeName)) {
            return
        }
    }

    $target = Find-ToolDirectory -ExeName $ExeName -Kind $Kind
    if (-not $target) {
        return
    }

    $alreadyPresent = $all | Where-Object { $_.TrimEnd('\') -ieq $target } | Select-Object -First 1
    if (-not $alreadyPresent) {
        $newUser = @($user) + @($target)
        [Environment]::SetEnvironmentVariable('Path', ($newUser -join ';'), 'User')
        Write-Host "Added to Windows User PATH: $target"

        $existingState = @()
        if (Test-Path $StateFile) {
            $existingState = Get-Content -Path $StateFile -ErrorAction SilentlyContinue | Where-Object { $_ -ne '' }
        }
        if (-not ($existingState | Where-Object { $_.TrimEnd('\') -ieq $target })) {
            Add-Content -Path $StateFile -Value $target -Encoding UTF8
        }
    }
}

function Remove-RecordedUserPaths {
    if (-not (Test-Path $StateFile)) {
        return
    }

    $recorded = @(
        Get-Content -Path $StateFile -ErrorAction SilentlyContinue |
            ForEach-Object { $_.Trim().TrimEnd('\') } |
            Where-Object { $_ -ne '' }
    )

    $user = @(Get-PathParts ([Environment]::GetEnvironmentVariable('Path', 'User')))
    if ($user.Count -gt 0 -and $recorded.Count -gt 0) {
        $kept = @(
            $user | Where-Object {
                $norm = $_.TrimEnd('\')
                -not ($recorded | Where-Object { $_ -ieq $norm })
            }
        )
        [Environment]::SetEnvironmentVariable('Path', ($kept -join ';'), 'User')
        foreach ($dir in $recorded) {
            Write-Host "Removed from Windows User PATH: $dir"
        }
    }

    Remove-Item -Path $StateFile -Force -ErrorAction SilentlyContinue
}

function Test-WingetPackageInstalled {
    param([string]$Id)
    $prevEAP = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    try {
        Write-Host "Checking installed package: $Id"
        & winget list --id $Id --exact --source winget --accept-source-agreements --disable-interactivity | Out-Host
        $code = $LASTEXITCODE
        if ($code -eq 0) { return $true }
        # APPINSTALLER_CLI_ERROR_NO_APPLICATIONS_FOUND (0x8A150014).
        if ($code -eq -1978335212) { return $false }
        throw "winget list failed for $Id (exit code $code). Check winget --info and winget --logs."
    }
    finally {
        $ErrorActionPreference = $prevEAP
    }
}

function Install-WingetPackage {
    param(
        [string]$Id,
        [string]$Version = ''
    )

    if ([string]::IsNullOrWhiteSpace($Version) -and (Test-WingetPackageInstalled -Id $Id)) {
        Write-Host "Already installed: $Id"
        return
    }

    $wingetArgs = @('install', '--id', $Id, '--exact', '--source', 'winget',
        '--accept-package-agreements', '--accept-source-agreements', '--silent',
        '--disable-interactivity', '--verbose-logs')
    if (-not [string]::IsNullOrWhiteSpace($Version)) {
        $wingetArgs += @('--version', $Version)
        Write-Host "Installing $Id ($Version)..."
    }
    else {
        Write-Host "Installing $Id..."
    }

    & winget @wingetArgs
    if ($LASTEXITCODE -ne 0) {
        throw "winget install failed for $Id (exit code $LASTEXITCODE)."
    }
}

function Uninstall-WingetPackage {
    param([string]$Id)

    if (-not (Test-WingetPackageInstalled -Id $Id)) {
        Write-Host "Not installed (skipping): $Id"
        return
    }

    Write-Host "Uninstalling $Id..."
    & winget uninstall --id $Id --exact --source winget --accept-source-agreements --silent --disable-interactivity
    if ($LASTEXITCODE -ne 0) {
        throw "winget uninstall failed for $Id (exit code $LASTEXITCODE)."
    }
}

function Invoke-CommandChecked {
    param(
        [string]$Command,
        [string[]]$Arguments
    )

    & $Command @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "'$Command $($Arguments -join ' ')' failed with exit code $LASTEXITCODE."
    }
}

function Invoke-OutputChecked {
    param([string]$Command, [string[]]$Arguments)
    $output = & $Command @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "'$Command $($Arguments -join ' ')' failed with exit code $LASTEXITCODE."
    }
    return ($output -join "`n").Trim()
}

function Get-NativeTreeSitter {
    $root = Invoke-OutputChecked -Command 'npm.cmd' -Arguments @('root', '--global')
    $exe = Join-Path $root 'tree-sitter-cli\tree-sitter.exe'
    if (-not (Test-Path -LiteralPath $exe -PathType Leaf)) {
        throw "Native Tree-sitter executable not found: $exe. Check the npm installation."
    }
    return $exe
}

function Initialize-Repository {
    Write-Host ''
    Write-Host '=== 4. Preparing Repository & Go Dependencies ==='
    $script:Checkout = $ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath($RepoDirectory)
    if (-not (Test-Path -LiteralPath $script:Checkout)) {
        $parent = Split-Path -Parent $script:Checkout
        New-Item -ItemType Directory -Path $parent -Force | Out-Null
        Invoke-CommandChecked -Command 'git' -Arguments @('clone', '--', $RepoUrl, $script:Checkout)
        if ($Revision) {
            Invoke-CommandChecked -Command 'git' -Arguments @('-C', $script:Checkout, 'checkout', '--detach', $Revision)
        } else {
            Write-Warning 'Cloned the default branch. Confirm it contains the fixes under test.'
        }
    } elseif ($Revision) {
        throw '-Revision only applies to a new clone. Use a new -RepoDirectory or select the revision yourself.'
    }
    $module = Join-Path $script:Checkout 'go.mod'
    if (-not (Test-Path -LiteralPath $module -PathType Leaf) -or
        -not (Test-Path -LiteralPath (Join-Path $script:Checkout 'cmd\ts-build\main.go') -PathType Leaf)) {
        throw "Not a go-tree-sitter source directory: $script:Checkout"
    }
    if ((Get-Content -LiteralPath $module -Raw) -notmatch '(?m)^module github\.com/michaelbomholt665/go-tree-sitter\s*$') {
        throw "Unexpected Go module in $module"
    }
    Push-Location -LiteralPath $script:Checkout
    try {
        Invoke-CommandChecked -Command 'go' -Arguments @('mod', 'download')
        Write-Host ''
        Write-Host '=== 5. Configuring Windows (Python only) ==='
        $dotenv = Join-Path $script:Checkout '.env'
        if (-not (Test-Path -LiteralPath $dotenv)) {
            [IO.File]::WriteAllText($dotenv, "TS_BUILD_ENV=DEVELOPMENT`r`n", [Text.Encoding]::ASCII)
        } else {
            Write-Host 'Preserving existing .env; test commands select their configuration explicitly.'
        }
        # Go's CC/CXX values quote executable paths; the app config stores raw paths.
        $cc = $env:CC.Trim('"')
        $cxx = $env:CXX.Trim('"')
        $gccVersion = Invoke-OutputChecked -Command $cc -Arguments @('-dumpfullversion', '-dumpversion')
        $gccTriple = Invoke-OutputChecked -Command $cc -Arguments @('-dumpmachine')
        if (-not $gccVersion -or -not $gccTriple) { throw 'GCC returned an empty version or target triple.' }
        $config = [ordered]@{
            version = '2.0'
            tree_sitter_cli_version = '0.27.0'
            abi_range = [ordered]@{ min = 13; max = 15 }
            generate_abi = 15
            build_dir = 'windows-vm-build'
            OS_TARGET = 'windows'
            targets = [ordered]@{
                windows = [ordered]@{
                    os = 'windows'; arch = 'amd64'; triple = $gccTriple
                    compiler = $cc; cxx = $cxx; compiler_version = $gccVersion
                }
            }
            output = [ordered]@{
                grammar_base = 'windows-vm-output'
                grammar_build = '{build_dir}/{lang}/{arch}'
                grammar_compile = '{build_dir}/{lang}/{arch}'
                generate_manifest = $true
            }
            runtime = [ordered]@{
                tools = [ordered]@{ tree_sitter = $script:TreeSitterExe }
                compiler = [ordered]@{ allow_fallback = $false }
            }
            languages = @([ordered]@{
                name = 'python'; grammar = 'python'; constructor = 'tree_sitter_python'
                version = 'v0.25.1-0.20250915213335-26855eabccb1'
                repository = 'https://github.com/tree-sitter/tree-sitter-python.git'
                revision = '26855eabccb19c6abf499fbc5b8dc7cc9ab8bc64'
                sample = "x = 1`n"
            })
        }
        # JSON is a YAML subset. Serialize paths rather than editing YAML with regex.
        $script:TestConfig = Join-Path $script:Checkout 'windows-vm-config.yaml'
        $json = $config | ConvertTo-Json -Depth 10
        if (Test-Path -LiteralPath $script:TestConfig) {
            if ((Get-Content -LiteralPath $script:TestConfig -Raw).Trim() -ne $json.Trim()) {
                throw "Preserving existing $script:TestConfig. Use a fresh source directory for different settings."
            }
        } else {
            [IO.File]::WriteAllText($script:TestConfig, $json, (New-Object Text.UTF8Encoding($false)))
        }
        # Validate the generated configuration through the real app before building a grammar.
        Invoke-CommandChecked -Command 'go' -Arguments @('run', './cmd/ts-build', '--config', $script:TestConfig, 'config', 'get', 'build_dir')
        Write-Host "Checkout: $script:Checkout"
        Write-Host 'Repository tree-sitter-config.yaml is unchanged; the test config pins the installed GCC version.'
        Write-Host 'To run the single-grammar pipeline manually, open PowerShell in the checkout and use:'
        Write-Host "`$env:CGO_ENABLED = '1'"
        Write-Host "`$env:CC = '$($env:CC)'"
        Write-Host "`$env:CXX = '$($env:CXX)'"
        Write-Host 'go run ./cmd/ts-build --config ./windows-vm-config.yaml build -l python'
        Write-Host 'go run ./cmd/ts-build --config ./windows-vm-config.yaml compile -l python --os windows --arch amd64'
        Write-Host 'go run ./cmd/ts-build --config ./windows-vm-config.yaml move -l python --compact --clean=false'
    } finally {
        Pop-Location
    }
}

function Invoke-PythonTest {
    if ([Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString() -ne 'X64') {
        throw 'The single-grammar VM test requires Windows x64.'
    }
    Push-Location -LiteralPath $script:Checkout
    try {
        if (Test-Path -LiteralPath 'windows-vm-output') {
            throw 'The test catalog already exists. Windows atomic replacement is unsupported; use a fresh test checkout.'
        }
        Write-Host ''
        Write-Host '=== 6. Building, Compiling & Publishing Python Only ==='
        foreach ($stage in @('build', 'compile', 'move')) {
            $arguments = @('run', './cmd/ts-build', '--config', $script:TestConfig, '--verbose', $stage, '--language', 'python')
            if ($stage -eq 'compile') { $arguments += @('--os', 'windows', '--arch', 'amd64') }
            if ($stage -eq 'move') { $arguments += @('--compact', '--clean=false') }
            Invoke-CommandChecked -Command 'go' -Arguments $arguments
        }
        Invoke-CommandChecked -Command 'go' -Arguments @('run', './cmd/ts-build', '--config', $script:TestConfig, 'compact', '--language', 'python', '--check')
        Write-Host 'Python native validation and publication passed. Results: windows-vm-output/python'
    } finally {
        Pop-Location
    }
}

function Invoke-Install {
    Assert-WingetAvailable

    Write-Host '=== 1. Installing Core Tooling (winget) ==='
    Install-WingetPackage -Id 'GoLang.Go'
    Install-WingetPackage -Id 'Git.Git'
    Install-WingetPackage -Id 'OpenJS.NodeJS.LTS'
    # tree-sitter-config.yaml expects x86_64-w64-mingw32-gcc 14.2.0 by default.
    # Pass -WinLibsVersion <exact-version> (from `winget show --id BrechtSanders.WinLibs.POSIX.UCRT --versions`)
    # if you want to pin a specific WinLibs release.
    Install-WingetPackage -Id 'BrechtSanders.WinLibs.POSIX.UCRT' -Version $WinLibsVersion

    Write-Host ''
    Write-Host '=== Configuring Windows User PATH ==='
    Ensure-ToolInUserPath -ExeName 'go.exe' -Kind 'go'
    Ensure-ToolInUserPath -ExeName 'git.exe' -Kind 'git'
    Ensure-ToolInUserPath -ExeName 'node.exe' -Kind 'node'
    Ensure-ToolInUserPath -ExeName 'x86_64-w64-mingw32-gcc.exe' -Kind 'winlibs'
    Refresh-ProcessPath

    foreach ($required in @('go', 'git', 'node', 'npm.cmd', 'x86_64-w64-mingw32-gcc')) {
        if (-not (Get-Command $required -ErrorAction SilentlyContinue)) {
            throw "'$required' is not visible on PATH yet. Close and reopen PowerShell, then re-run this script."
        }
    }

    Write-Host ''
    Write-Host '=== 2. Installing Pinned tree-sitter CLI (0.27.0) ==='
    Invoke-CommandChecked -Command 'npm.cmd' -Arguments @('install', '-g', 'tree-sitter-cli@0.27.0')

    Ensure-ToolInUserPath -ExeName 'tree-sitter.cmd' -Kind 'npm-global'
    Refresh-ProcessPath

    Write-Host ''
    Write-Host '=== 3. Verifying All Tools & Enabling CGo ==='
    Invoke-CommandChecked -Command 'go' -Arguments @('version')                 # must be 1.26.0 or higher
    Invoke-CommandChecked -Command 'git' -Arguments @('--version')
    Invoke-CommandChecked -Command 'node' -Arguments @('--version')
    Invoke-CommandChecked -Command 'npm.cmd' -Arguments @('--version')
    $script:TreeSitterExe = Get-NativeTreeSitter
    Invoke-CommandChecked -Command $script:TreeSitterExe -Arguments @('--version')
    Invoke-CommandChecked -Command 'x86_64-w64-mingw32-gcc' -Arguments @('--version')
    Invoke-CommandChecked -Command 'go' -Arguments @('env', '-w', 'CGO_ENABLED=1')
    $env:CGO_ENABLED = '1'
    $cc = (Get-Command x86_64-w64-mingw32-gcc.exe -CommandType Application -ErrorAction Stop).Source
    $cxx = (Get-Command x86_64-w64-mingw32-g++.exe -CommandType Application -ErrorAction Stop).Source
    $env:CC = '"' + $cc + '"'
    $env:CXX = '"' + $cxx + '"'

    Write-Host ''
    Write-Host 'All prerequisites installed, PATH updated, and CGO_ENABLED=1 set.'
    Initialize-Repository
    if ($RunGrammarTest) { Invoke-PythonTest }
}

function Invoke-Uninstall {
    Assert-WingetAvailable
    Write-Warning 'This action removes Go, Git, Node.js, and WinLibs, including installations that predate this script.'
    if ((Read-Host 'Type UNINSTALL to continue') -cne 'UNINSTALL') { throw 'Uninstall canceled.' }
    Refresh-ProcessPath

    Write-Host '=== 1. Removing tree-sitter CLI & CGo setting ==='
    $prevEAP = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    try {
        if (Get-Command npm.cmd -ErrorAction SilentlyContinue) {
            & npm.cmd uninstall -g tree-sitter-cli
            if ($LASTEXITCODE -ne 0) { throw 'npm uninstall failed; core tools were not removed.' }
        }
        if (Get-Command go -ErrorAction SilentlyContinue) {
            & go env -u CGO_ENABLED *> $null
        }
    }
    finally {
        $ErrorActionPreference = $prevEAP
    }

    Write-Host ''
    Write-Host '=== 2. Uninstalling Core Tooling (winget) ==='
    for ($i = $WingetPackages.Count - 1; $i -ge 0; $i--) {
        Uninstall-WingetPackage -Id $WingetPackages[$i]
    }

    Write-Host ''
    Write-Host '=== 3. Cleaning Added Windows User PATH Entries ==='
    Remove-RecordedUserPaths
    Refresh-ProcessPath

    Write-Host ''
    Write-Host 'Uninstall complete. Open a new PowerShell window if any shell cache still holds old paths.'
}

switch ($Action) {
    'install'   { Invoke-Install }
    'uninstall' { Invoke-Uninstall }
    'help'      { Show-Usage }
}
