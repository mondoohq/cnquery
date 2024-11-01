// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package packages

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseWindowsAppxManifest(t *testing.T) {
	manifest := `<?xml version="1.0" encoding="utf-8"?>
<Package
  xmlns="http://schemas.microsoft.com/appx/manifest/foundation/windows10" 
  xmlns:uap="http://schemas.microsoft.com/appx/manifest/uap/windows10"
  xmlns:rescap="http://schemas.microsoft.com/appx/manifest/foundation/windows10/restrictedcapabilities"
  xmlns:wincap="http://schemas.microsoft.com/appx/manifest/foundation/windows10/windowscapabilities"
  IgnorableNamespaces="uap rescap wincap">

  <!--
  Manual versioning is used for this app.
  Appx version should be in sync with version used for the app name in microsoft-windows-diagnosticcomposerhost.appxsetup.man        
  See https://osgwiki.com/wiki/System_Apps#Servicing
  -->
  <Identity Name="Microsoft.AAD.BrokerPlugin"
          Publisher="CN=Microsoft Windows, O=Microsoft Corporation, L=Redmond, S=Washington, C=US"
          ProcessorArchitecture="neutral"
          ResourceId="neutral"
          Version="1000.19580.1000.0" />

  <Properties>
    <DisplayName>ms-resource:PackageDisplayName</DisplayName>
    <PublisherDisplayName>ms-resource:PublisherDisplayName</PublisherDisplayName>
    <Logo>Assets\StoreLogo.png</Logo>
  </Properties>

  <Resources>
    <Resource Language="en-us" />
  </Resources>
  <Applications>
    <Application Id="App" Executable="Microsoft.AAD.BrokerPlugin.exe" EntryPoint="BrokerPlugin.App">
      <uap:VisualElements DisplayName="ms-resource:PackageDisplayName" Square150x150Logo="Assets\Logo.png" Square44x44Logo="Assets\SmallLogo.png" Description="ms-resource:PackageDescription" BackgroundColor="#ffffff" AppListEntry="none">
        <uap:SplashScreen Image="Assets\SplashScreen.png" />
      </uap:VisualElements>
      <Extensions>
        <uap:Extension Category="windows.webAccountProvider">
          <uap:WebAccountProvider Url="https://login.windows.net" BackgroundEntryPoint="AAD.Core.TokenBackground" />
        </uap:Extension>
        <uap:Extension Category="windows.appService" EntryPoint="AAD.Core.AppService">
          <uap:AppService Name="TBAuthAppService" />
        </uap:Extension>
        <uap:Extension Category="windows.protocol">
          <uap:Protocol Name="ms-aad-brokerplugin">
            <uap:DisplayName>ms-resource:PackageDisplayName</uap:DisplayName>
          </uap:Protocol>
        </uap:Extension>
      </Extensions>
    </Application>
  </Applications>
  <Dependencies>
    <TargetDeviceFamily Name="Windows.Universal" MinVersion="10.0.0.0" MaxVersionTested="10.0.10587.0"/>
  </Dependencies>
  <Capabilities>
    <Capability Name="internetClient" />
    <uap:Capability Name="enterpriseAuthentication" />
    <Capability Name="privateNetworkClientServer" />
    <uap:Capability Name="sharedUserCertificates" />
    <rescap:Capability Name="deviceManagementAdministrator" />
    <rescap:Capability Name="deviceManagementRegistration" />
    <rescap:Capability Name="remotePassportAuthentication" />
    <rescap:Capability Name="userPrincipalName" />
    <rescap:Capability Name="windowsHelloCredentialAccess" />
    <!-- needed for detection of Visitor accounts using Windows::System::Internal::UserManager::GetProfileForUser -->
    <wincap:Capability Name="userSigninSupport" />
  </Capabilities>
  <Extensions>
    <Extension Category="windows.activatableClass.inProcessServer">
      <InProcessServer>
        <Path>AAD.Core.dll</Path>
        <!-- Value of ActivatableClassId should be the same as BackgroundEntryPoint
             value in WebAccountProvider extension.
          -->
        <ActivatableClass ActivatableClassId="AAD.Core.TokenBackground" ThreadingModel="both" />
        <ActivatableClass ActivatableClassId="AAD.Core.AppService" ThreadingModel="both" />
        <ActivatableClass ActivatableClassId="AAD.Core.WebAccountProcessor" ThreadingModel="both" />
      </InProcessServer>
    </Extension>
  </Extensions>

</Package>`

	man, err := parseAppxManifest([]byte(manifest))
	require.NoError(t, err)

	require.Equal(t, "neutral", man.arch)
	require.Equal(t, "Microsoft.AAD.BrokerPlugin", man.Name)
	require.Equal(t, "CN=Microsoft Windows, O=Microsoft Corporation, L=Redmond, S=Washington, C=US", man.Publisher)
	require.Equal(t, "1000.19580.1000.0", man.Version)
}
