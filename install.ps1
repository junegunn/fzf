$version="0.42.0"

$fzf_base=Split-Path -Parent $MyInvocation.MyCommand.Definition

function check_binary () {
  Write-Host "  - Checking fzf executable ... " -NoNewline
  $output=cmd /c $fzf_base\bin\fzf.exe --version 2>&1
  if (-not $?) {
    Write-Host "Error: $output"
    $binary_error="Invalid binary"
  } else {
    $output=(-Split $output)[0]
    if ($version -ne $output) {
      Write-Host "$output != $version"
      $binary_error="Invalid version"
    } else {
      Write-Host "$output"
      $binary_error=""
      return 1
    }
  }
  Remove-Item "$fzf_base\bin\fzf.exe"
  return 0
}

function download {
  param($file)
  Write-Host "Downloading bin/fzf ..."
  if (Test-Path "$fzf_base\bin\fzf.exe") {
    Write-Host "  - Already exists"
    if (check_binary) {
      return
    }
  }
  if (-not (Test-Path "$fzf_base\bin")) {
    md "$fzf_base\bin"
  }
  if (-not $?) {
    $binary_error="Failed to create bin directory"
    return
  }
  cd "$fzf_base\bin"
  $url="https://github.com/junegunn/fzf/releases/download/$version/$file"
  $temp=$env:TMP + "\fzf.zip"
  [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
  if ($PSVersionTable.PSVersion.Major -ge 3) {
    Invoke-WebRequest -Uri $url -OutFile $temp
  } else {
    (New-Object Net.WebClient).DownloadFile($url, $ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath("$temp"))
  }
  if ($?) {
    (Microsoft.PowerShell.Archive\Expand-Archive -Path $temp -DestinationPath .); (Remove-Item $temp)
  } else {
    $binary_error="Failed to download with powershell"
  }
  if (-not (Test-Path fzf.exe)) {
    $binary_error="Failed to download $file"
    return
  }
  echo y | icacls $fzf_base\bin\fzf.exe /grant Administrator:F ; check_binary >$null
}

download "fzf-$version-windows_amd64.zip"

Write-Host 'For more information, see: https://github.com/junegunn/fzf'
