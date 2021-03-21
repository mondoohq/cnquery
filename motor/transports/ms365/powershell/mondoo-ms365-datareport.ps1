# .\exchangeonline.ps1 |  Out-File mondoo-ms365-datareport.json
# Set-ExecutionPolicy RemoteSigned
# $UserCredential = Get-Credential

# Install-Module -Name ExchangeOnlineManagement -Scope CurrentUser -Force
# Import-Module ExchangeOnlineManagement
# Connect-ExchangeOnline -Credential $UserCredential
# Disconnect-ExchangeOnline

# https://docs.microsoft.com/en-us/powershell/module/sharepoint-online/?view=sharepoint-ps
# Install-Module -Name Microsoft.Online.SharePoint.PowerShell -Scope CurrentUser -Force
# Import-Module SharePointOnlinePowerShell
# Connect-SPOService -Url https://tenant-admin.sharepoint.com -Credential $UserCredential
# Disconnect-SPOService

# https://docs.microsoft.com/en-us/MicrosoftTeams/teams-powershell-overview
# Install-Module MicrosoftTeams -Scope CurrentUser -Force
# Import-Module MicrosoftTeams
# Connect-MicrosoftTeams -Credential $UserCredential
# Disconnect-MicrosoftTeams

$MalwareFilterPolicy = (Get-MalwareFilterPolicy)
$HostedOutboundSpamFilterPolicy = (Get-HostedOutboundSpamFilterPolicy)
$TransportRule = (Get-TransportRule)
$RemoteDomain = (Get-RemoteDomain Default)
$SafeLinksPolicy = (Get-SafeLinksPolicy)
$SafeAttachmentPolicy = (Get-SafeAttachmentPolicy)
$OrganizationConfig = (Get-OrganizationConfig)
$AuthenticationPolicy = (Get-AuthenticationPolicy)
$AntiPhishPolicy = (Get-AntiPhishPolicy)
$DkimSigningConfig = (Get-DkimSigningConfig)
$OwaMailboxPolicy = (Get-OwaMailboxPolicy)
$AdminAuditLogConfig = (Get-AdminAuditLogConfig)
$PhishFilterPolicy = (Get-PhishFilterPolicy)
$Mailbox = (Get-Mailbox -ResultSize Unlimited)
$AtpPolicyForO365 = (Get-AtpPolicyForO365)
$SharingPolicy = (Get-SharingPolicy)
$RoleAssignmentPolicy = (Get-RoleAssignmentPolicy)

# collect exchange online data
$exchangeOnline = New-Object PSObject
Add-Member -InputObject $exchangeOnline -MemberType NoteProperty -Name MalwareFilterPolicy -Value @($MalwareFilterPolicy)
Add-Member -InputObject $exchangeOnline -MemberType NoteProperty -Name HostedOutboundSpamFilterPolicy -Value @($HostedOutboundSpamFilterPolicy)
Add-Member -InputObject $exchangeOnline -MemberType NoteProperty -Name TransportRule -Value @($TransportRule)
Add-Member -InputObject $exchangeOnline -MemberType NoteProperty -Name RemoteDomain -Value  @($RemoteDomain)
Add-Member -InputObject $exchangeOnline -MemberType NoteProperty -Name SafeLinksPolicy -Value @($SafeLinksPolicy)
Add-Member -InputObject $exchangeOnline -MemberType NoteProperty -Name SafeAttachmentPolicy -Value @($SafeAttachmentPolicy)
Add-Member -InputObject $exchangeOnline -MemberType NoteProperty -Name OrganizationConfig -Value $OrganizationConfig
Add-Member -InputObject $exchangeOnline -MemberType NoteProperty -Name AuthenticationPolicy -Value @($AuthenticationPolicy)
Add-Member -InputObject $exchangeOnline -MemberType NoteProperty -Name AntiPhishPolicy -Value @($AntiPhishPolicy)
Add-Member -InputObject $exchangeOnline -MemberType NoteProperty -Name DkimSigningConfig -Value @($DkimSigningConfig)
Add-Member -InputObject $exchangeOnline -MemberType NoteProperty -Name OwaMailboxPolicy -Value @($OwaMailboxPolicy)
Add-Member -InputObject $exchangeOnline -MemberType NoteProperty -Name AdminAuditLogConfig -Value $AdminAuditLogConfig
Add-Member -InputObject $exchangeOnline -MemberType NoteProperty -Name PhishFilterPolicy -Value @($PhishFilterPolicy)
Add-Member -InputObject $exchangeOnline -MemberType NoteProperty -Name Mailbox -Value @($Mailbox)
Add-Member -InputObject $exchangeOnline -MemberType NoteProperty -Name AtpPolicyForO365 -Value @($AtpPolicyForO365)
Add-Member -InputObject $exchangeOnline -MemberType NoteProperty -Name SharingPolicy -Value @($SharingPolicy)
Add-Member -InputObject $exchangeOnline -MemberType NoteProperty -Name RoleAssignmentPolicy -Value @($RoleAssignmentPolicy)

# collect sharepoint data

$SPOTenant = (Get-SPOTenant)
$SPOTenantSyncClientRestriction = (Get-SPOTenantSyncClientRestriction)

$sharepointOnline = New-Object PSObject
Add-Member -InputObject $sharepointOnline -MemberType NoteProperty -Name SPOTenant -Value $SPOTenant
Add-Member -InputObject $sharepointOnline -MemberType NoteProperty -Name SPOTenantSyncClientRestriction -Value $SPOTenantSyncClientRestriction

# collect msteams data
$CsTeamsClientConfiguration = (Get-CsTeamsClientConfiguration)
$CsOAuthConfiguration = (Get-CsOAuthConfiguration)


$msteams = New-Object PSObject
Add-Member -InputObject $msteams -MemberType NoteProperty -Name CsTeamsClientConfiguration -Value $CsTeamsClientConfiguration
Add-Member -InputObject $msteams -MemberType NoteProperty -Name CsOAuthConfiguration -Value @($CsOAuthConfiguration)

# generate report
$report = New-Object PSObject
Add-Member -InputObject $report -MemberType NoteProperty -Name ExchangeOnline -Value $exchangeOnline
Add-Member -InputObject $report -MemberType NoteProperty -Name SharepointOnline -Value $sharepointOnline
Add-Member -InputObject $report -MemberType NoteProperty -Name MsTeams -Value $msteams

ConvertTo-Json -Depth 4 $report