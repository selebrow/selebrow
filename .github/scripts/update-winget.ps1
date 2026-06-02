param(
    [Parameter(Mandatory = $true)]
    [string]$Tag,

    [Parameter(Mandatory = $true)]
    [string]$ForkOwner,

    [switch]$DryRun,

    [switch]$Force
)

$ErrorActionPreference = "Stop"

$packageId = "SelebrowProject.Selebrow"
$packageRoot = "manifests/s/SelebrowProject/Selebrow"
$version = $Tag.TrimStart("v")

if (-not $DryRun -and -not $env:WINGET_PAT) {
    throw "WINGET_PAT secret is not set"
}

Write-Host "Tag: $Tag"
Write-Host "Version: $version"
Write-Host "PackageId: $packageId"
Write-Host "Fork owner: $ForkOwner"
Write-Host "DryRun: $DryRun"
Write-Host "Force: $Force"

$releaseUrl = "https://api.github.com/repos/selebrow/selebrow/releases/tags/$Tag"

$release = Invoke-RestMethod `
    -Uri $releaseUrl `
    -Headers @{ "User-Agent" = "github-actions" }

$amd64Asset = $release.assets |
    Where-Object { $_.name -eq "selebrow-windows-amd64.exe" } |
    Select-Object -First 1

$arm64Asset = $release.assets |
    Where-Object { $_.name -eq "selebrow-windows-arm64.exe" } |
    Select-Object -First 1

if (-not $amd64Asset) {
    throw "selebrow-windows-amd64.exe not found in release assets"
}

if (-not $arm64Asset) {
    throw "selebrow-windows-arm64.exe not found in release assets"
}

$amd64Url = $amd64Asset.browser_download_url
$arm64Url = $arm64Asset.browser_download_url

Write-Host "amd64 URL: $amd64Url"
Write-Host "arm64 URL: $arm64Url"

Invoke-WebRequest $amd64Url -OutFile "selebrow-windows-amd64.exe"
Invoke-WebRequest $arm64Url -OutFile "selebrow-windows-arm64.exe"

$amd64Hash = (Get-FileHash "selebrow-windows-amd64.exe" -Algorithm SHA256).Hash.ToLower()
$arm64Hash = (Get-FileHash "selebrow-windows-arm64.exe" -Algorithm SHA256).Hash.ToLower()

Write-Host "amd64 SHA256: $amd64Hash"
Write-Host "arm64 SHA256: $arm64Hash"

git config --global user.name "selebrow-bot"
git config --global user.email "selebrow.dev+bot@gmail.com"

$branch = "SelebrowProject.Selebrow-$version"

$cloneUrl = if ($DryRun) {
    "https://github.com/$ForkOwner/winget-pkgs.git"
} else {
    "https://x-access-token:$env:WINGET_PAT@github.com/$ForkOwner/winget-pkgs.git"
}

Remove-Item "winget-pkgs" -Recurse -Force -ErrorAction SilentlyContinue

git clone `
    --depth 1 `
    --filter=blob:none `
    --sparse `
    $cloneUrl

Set-Location "winget-pkgs"

git sparse-checkout set $packageRoot

git remote add upstream "https://github.com/microsoft/winget-pkgs.git" 2>$null

git fetch `
    --depth 1 `
    upstream `
    master

git checkout -B $branch FETCH_HEAD

if (-not (Test-Path $packageRoot)) {
    throw "Package root not found: $packageRoot"
}

$targetPath = "$packageRoot/$version"

if (Test-Path $targetPath) {
    if ($Force) {
        Write-Host "Version folder already exists locally. Reusing it because -Force is enabled: $targetPath"

        $oldVersion = $version
        $newPath = $targetPath
    } else {
        throw "Version $version already exists. Use -Force for local dry-run testing."
    }
} else {
    $previousVersion = Get-ChildItem $packageRoot -Directory |
        Sort-Object { [version]$_.Name } -Descending |
        Select-Object -First 1

    if (-not $previousVersion) {
        throw "Previous package version not found"
    }

    $oldVersion = $previousVersion.Name

    Write-Host "Previous version: $oldVersion"
    Write-Host "New version: $version"

    $oldPath = "$packageRoot/$oldVersion"
    $newPath = "$packageRoot/$version"

    Copy-Item $oldPath $newPath -Recurse
}

$versionFile = "$newPath/SelebrowProject.Selebrow.yaml"
$localeFile = "$newPath/SelebrowProject.Selebrow.locale.en-US.yaml"
$installerFile = "$newPath/SelebrowProject.Selebrow.installer.yaml"

$utf8NoBom = [System.Text.UTF8Encoding]::new($false)

$versionContent = (Get-Content $versionFile -Raw) `
    -replace "PackageVersion: $oldVersion", "PackageVersion: $version"

[System.IO.File]::WriteAllText(
    (Resolve-Path $versionFile),
    $versionContent.TrimEnd() + [Environment]::NewLine,
    $utf8NoBom
)

$localeContent = (Get-Content $localeFile -Raw) `
    -replace "PackageVersion: $oldVersion", "PackageVersion: $version"

[System.IO.File]::WriteAllText(
    (Resolve-Path $localeFile),
    $localeContent.TrimEnd() + [Environment]::NewLine,
    $utf8NoBom
)

$installerContent = @"
# yaml-language-server: `$schema=https://aka.ms/winget-manifest.installer.1.12.0.schema.json

PackageIdentifier: SelebrowProject.Selebrow
PackageVersion: $version
InstallerType: portable
Commands:
  - selebrow
Installers:
  - Architecture: x64
    InstallerUrl: $amd64Url
    InstallerSha256: $amd64Hash
  - Architecture: arm64
    InstallerUrl: $arm64Url
    InstallerSha256: $arm64Hash
ManifestType: installer
ManifestVersion: 1.12.0
"@

[System.IO.File]::WriteAllText(
    (Resolve-Path $installerFile),
    $installerContent.TrimEnd() + [Environment]::NewLine,
    $utf8NoBom
)

Write-Host "Generated manifest files:"
Get-ChildItem $newPath

Write-Host "Generated manifest path:"
Write-Host $newPath

winget validate --manifest $newPath

if ($DryRun) {
    Write-Host "Dry run enabled. Skipping commit, push and PR creation."
    Write-Host "Generated files are here:"
    Write-Host $newPath
    return
}

git status

git add $newPath
git commit -m "New version: SelebrowProject.Selebrow version $version"
git push --force origin $branch

gh pr create `
    --repo microsoft/winget-pkgs `
    --base master `
    --head "${ForkOwner}:$branch" `
    --title "New version: SelebrowProject.Selebrow version $version" `
    --body "New version: SelebrowProject.Selebrow version $version"
