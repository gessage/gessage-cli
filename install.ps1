$version = "v1.0.0"
$binary = "gessage"
$arch = if ([Environment]::Is64BitProcess) { "amd64" } else { "x86" }
$os = "windows"

$url = "https://github.com/ispooya/gessage-cli/releases/download/$version/$binary-$os-$arch.exe"
$destination = "$env:USERPROFILE\$binary.exe"

Write-Host "Downloading $url..."
Invoke-WebRequest -Uri $url -OutFile $destination
Write-Host "$binary installed to $destination"

# Optional: Add to PATH
Write-Host "Add $destination to your PATH to run from anywhere."