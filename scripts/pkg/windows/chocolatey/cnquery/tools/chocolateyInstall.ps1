$ErrorActionPreference = 'Stop'; # stop on all errors
$toolsDir   = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"

$version  = '7.0.3'
$url      = "https://releases.mondoo.com/cnquery/${version}/cnquery_${version}_windows_amd64.zip"
$checksum = '886c778d8368d71de18efb60f3cb24032c475e8c3eaabdab197597e36ed1130f'

$packageArgs = @{
  packageName   = $env:ChocolateyPackageName
  unzipLocation = $toolsDir
  url64bit      = $url

  checksum64    = $checksum
  checksumType64= 'sha256' #default is checksumType
}

Install-ChocolateyZipPackage @packageArgs 


