[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'

$repositoryRoot = Split-Path -Parent $PSScriptRoot
$excludedDirectories = @('.git', '.idea', '.tmp', '.vscode', 'build', 'coverage', 'dist')
$textExtensions = @(
    '.go', '.md', '.mod', '.sum', '.txt', '.yml', '.yaml', '.json',
    '.toml', '.xml', '.ps1', '.sh'
)
$prohibitedPatterns = @(
    '(?i)\b(?:convert|pars|splitt|export)er\b',
    '(?i)(?:pars|splitt|export)er:',
    '(?i)(?:commonSlice|structRecursion|titleRange)(?:Pars|Splitt|Export)er'
)

$violations = foreach ($file in Get-ChildItem -LiteralPath $repositoryRoot -Recurse -File) {
    $relativePath = [IO.Path]::GetRelativePath($repositoryRoot, $file.FullName)
    if ($file.FullName -eq $PSCommandPath) {
        continue
    }
    $segments = $relativePath -split '[\\/]'
    if ($segments | Where-Object { $_ -in $excludedDirectories }) {
        continue
    }
    if ($file.Extension -notin $textExtensions) {
        continue
    }

    $lineNumber = 0
    foreach ($line in Get-Content -LiteralPath $file.FullName) {
        $lineNumber++
        foreach ($pattern in $prohibitedPatterns) {
            if ($line -match $pattern) {
                [PSCustomObject]@{
                    Path = $relativePath
                    Line = $lineNumber
                }
                break
            }
        }
    }
}

if ($violations) {
    $violations | Sort-Object Path, Line | Format-Table -AutoSize | Out-String | Write-Host
    throw 'Prohibited legacy or application-specific terminology was found.'
}

Write-Host 'Sensitive-content scan passed.'
