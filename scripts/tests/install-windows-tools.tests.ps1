# Run with: pwsh -NoProfile -File scripts/tests/install-windows-tools.tests.ps1
# Mocked checks: no package installation, network access, or grammar compilation.
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$temp = Join-Path ([IO.Path]::GetTempPath()) ('ts-windows-installer-test-' + [guid]::NewGuid())
New-Item -ItemType Directory -Path $temp | Out-Null
$oldProfile = $env:USERPROFILE
$oldCC = $env:CC
$oldCXX = $env:CXX
try {
    $env:USERPROFILE = $temp
    . (Join-Path $root 'install-windows-tools.ps1') -Action help
    function Assert-True($Condition, [string]$Message) {
        if (-not $Condition) { throw $Message }
    }
    $script:Calls = @()
    $script:WingetExit = 0
    function winget {
        $script:Calls += ,@($args)
        if ($args[0] -eq 'list') {
            foreach ($required in @('--source', 'winget', '--accept-source-agreements', '--disable-interactivity')) {
                if ($args -notcontains $required) { throw "Hidden winget prompt risk: missing $required" }
            }
            $global:LASTEXITCODE = $script:WingetExit
        } else {
            $global:LASTEXITCODE = 0
        }
    }
    Assert-True (Test-WingetPackageInstalled 'GoLang.Go') 'Installed package was not detected.'
    $script:WingetExit = -1978335212
    Assert-True (-not (Test-WingetPackageInstalled 'GoLang.Go')) 'Missing package was treated as installed.'
    Install-WingetPackage -Id 'GoLang.Go'
    $installCall = $script:Calls[-1]
    foreach ($flag in @('--silent', '--disable-interactivity', '--accept-source-agreements', '--accept-package-agreements', '--source', 'winget')) {
        Assert-True ($installCall -contains $flag) "Install missing $flag"
    }
    $script:WingetExit = 42
    $failed = $false
    try { Test-WingetPackageInstalled 'GoLang.Go' | Out-Null } catch { $failed = $true }
    Assert-True $failed 'A winget query failure was treated as a missing package.'

    # Exercise real setup/config generation against a temporary source copy.
    $RepoDirectory = Join-Path $temp 'existing-checkout'
    New-Item -ItemType Directory -Path (Join-Path $RepoDirectory 'cmd/ts-build') -Force | Out-Null
    [IO.File]::WriteAllText((Join-Path $RepoDirectory 'go.mod'), "module github.com/michaelbomholt665/go-tree-sitter`n")
    [IO.File]::WriteAllText((Join-Path $RepoDirectory 'cmd/ts-build/main.go'), 'package main')
    [IO.File]::WriteAllText((Join-Path $RepoDirectory 'tree-sitter-config.yaml'), 'preserve repository config')
    $Revision = ''
    $script:TreeSitterExe = 'C:/tools/tree-sitter.exe'
    $env:CC = 'test-gcc'
    $env:CXX = 'test-g++'
    function test-gcc {
        $global:LASTEXITCODE = 0
        if ($args -contains '-dumpmachine') { 'x86_64-w64-mingw32' } else { '15.2.0' }
    }
    $script:GoExit = 0
    function go {
        $script:Calls += ,(@('go') + @($args))
        $global:LASTEXITCODE = $script:GoExit
    }
    function git {
        $script:Calls += ,(@('git') + @($args))
        if ($args[0] -eq 'clone') {
            $dest = $args[-1]
            New-Item -ItemType Directory -Path (Join-Path $dest 'cmd/ts-build') -Force | Out-Null
            [IO.File]::WriteAllText((Join-Path $dest 'go.mod'), "module github.com/michaelbomholt665/go-tree-sitter`n")
            [IO.File]::WriteAllText((Join-Path $dest 'cmd/ts-build/main.go'), 'package main')
        }
        $global:LASTEXITCODE = 0
    }
    $originalLocation = (Get-Location).Path
    Initialize-Repository
    Assert-True ((Get-Location).Path -eq $originalLocation) 'Setup did not restore the working directory.'
    $absoluteCheckout = $RepoDirectory
    Push-Location -LiteralPath $temp
    try {
        $RepoDirectory = './existing-checkout'
        Initialize-Repository
        Assert-True ($script:Checkout -eq $absoluteCheckout) 'Setup resolved a relative path against the wrong directory.'
    } finally {
        $RepoDirectory = $absoluteCheckout
        Pop-Location
    }
    $config = Get-Content -LiteralPath $script:TestConfig -Raw | ConvertFrom-Json
    Assert-True (@($config.languages).Count -eq 1) 'Test config selects more than one grammar.'
    Assert-True ($config.languages[0].name -eq 'python') 'Test config does not select Python.'
    Assert-True ($config.targets.windows.compiler_version -eq '15.2.0') 'Installed GCC version was not pinned.'
    Assert-True ($config.runtime.tools.tree_sitter -eq $script:TreeSitterExe) 'Native CLI path was not configured.'
    Assert-True ($config.output.grammar_base -eq 'windows-vm-output') 'Production catalog is not isolated.'
    Assert-True ((Get-Content (Join-Path $RepoDirectory 'tree-sitter-config.yaml') -Raw) -eq 'preserve repository config') 'Setup changed the repository config.'
    Assert-True ((Get-Content (Join-Path $RepoDirectory '.env') -Raw).Trim() -eq 'TS_BUILD_ENV=DEVELOPMENT') 'Development marker was not created.'
    $script:GoExit = 23
    $failed = $false
    try { Initialize-Repository } catch { $failed = $true }
    Assert-True $failed 'Setup ignored a Go dependency download failure.'
    Assert-True ((Get-Location).Path -eq $originalLocation) 'Failed setup did not restore the working directory.'
    $script:GoExit = 0
    [IO.File]::WriteAllText((Join-Path $RepoDirectory '.env'), "OTHER=value`nTS_BUILD_ENV=PRODUCTION`n")
    Initialize-Repository
    Assert-True ((Get-Content (Join-Path $RepoDirectory '.env') -Raw) -match 'OTHER=value') 'Existing .env was overwritten.'
    $originalConfig = Get-Content -LiteralPath $script:TestConfig -Raw
    [IO.File]::WriteAllText($script:TestConfig, 'preserve different settings')
    $failed = $false
    try { Initialize-Repository } catch { $failed = $true }
    Assert-True $failed 'Setup overwrote a different existing VM configuration.'
    Assert-True ((Get-Content -LiteralPath $script:TestConfig -Raw) -eq 'preserve different settings') 'Different VM configuration was changed.'
    [IO.File]::WriteAllText($script:TestConfig, $originalConfig)
    $env:CC = '"test-gcc"'
    $env:CXX = '"test-g++"'
    Initialize-Repository
    $quotedConfig = Get-Content -LiteralPath $script:TestConfig -Raw | ConvertFrom-Json
    Assert-True ($quotedConfig.targets.windows.compiler -eq 'test-gcc') 'Quoted Go compiler paths leaked into the app configuration.'
    $script:Calls = @()
    Invoke-PythonTest
    $buildCalls = @($script:Calls | Where-Object { $_ -contains '--language' })
    Assert-True ($buildCalls.Count -eq 4) 'Expected three pipeline stages and one compact check.'
    foreach ($call in $buildCalls) {
        Assert-True ($call -contains 'python') 'Pipeline selected a grammar other than Python.'
        Assert-True ($call -notcontains '--force') 'Pipeline attempted unsupported forced publication.'
    }
    Assert-True (@($script:Calls | Where-Object { $_ -contains 'test' }).Count -eq 0) 'Pipeline invoked the Go test suite.'
    New-Item -ItemType Directory -Path (Join-Path $RepoDirectory 'windows-vm-output') | Out-Null
    $failed = $false
    try { Invoke-PythonTest } catch { $failed = $true }
    Assert-True $failed 'Pipeline accepted an existing Windows catalog.'

    $RepoDirectory = Join-Path $temp 'new-checkout'
    $Revision = 'test-project-revision'
    $script:Calls = @()
    Initialize-Repository
    Assert-True (@($script:Calls | Where-Object { $_ -contains 'clone' }).Count -eq 1) 'New checkout was not cloned.'
    Assert-True (@($script:Calls | Where-Object { $_ -contains '--detach' }).Count -eq 1) 'New checkout ignored the requested revision.'
    $failed = $false
    try { Initialize-Repository } catch { $failed = $true }
    Assert-True $failed 'Setup switched an existing checkout to the requested revision.'
    Write-Output 'PASS: winget prompt guards, clone/reuse, config generation, and Python-only pipeline.'
} finally {
    $env:USERPROFILE = $oldProfile
    $env:CC = $oldCC
    $env:CXX = $oldCXX
    Remove-Item -LiteralPath $temp -Recurse -Force
}
