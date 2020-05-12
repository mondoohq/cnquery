import * as React from 'react';
import styled, { ThemeProvider } from 'styled-components';

import snapshots from '../../static/snapshots.json';
import { JsxElement } from 'typescript';


type LrArg = {
  ID: string
  Type: LrType
}

type LrSimpleArg = {
  Type: string
}

type LrType = {
  ListType?: {
    Type: LrType
  }
  MapType?: {
    Key: LrSimpleArg
    Value: LrType
  }
  SimpleType?: {
    Type: string
  }    
}

type LrField = {
  Args?: {
    List?: LrSimpleArg[]
  }
  ID: string
  Type: LrType
  updated?: boolean
}

type LrListResource = {
  Type: {
    Type: string
  }
}

type LrInit = {
  Args: LrArg[]
}

type LrResource = {
  Body: {
    Fields: LrField[] | null
    Inits: LrInit[] | null
  }
  ID: string
  ListType?: LrListResource
  updated?: boolean
}

type LrSnapshot = {
  version: string
  Resources: LrResource[]
}

type LrDocsState = {
  metadata?: LrSnapshot[]
  selected?: LrSnapshot
}
export class LrDocs extends React.Component<{}, LrDocsState> {
  state: LrDocsState = {}

  render() {
    if (this.state.metadata == null) {
      this.load_metadata()
      return "loading..."
    }

    let versions = this.state.metadata.map(x => x.version);
    let { selected } = this.state;

    let idx = versions.indexOf(selected.version);
    let prev = selected;
    if (idx <= versions.length-2) {
      prev = this.state.metadata[idx+1]
    }

    return (
      <SiteStructure>
        <Versions
          versions={versions}
          selected={selected.version}
          onSelect={(version) => {
            let selected = this.state.metadata.find(v => v.version == version)
            this.setState({
              selected,
            })
          }}
        />
        <Resources snapshot={selected} prev={prev} />
      </SiteStructure>
    )
  }

  load_metadata() {
    let data = snapshots;
    let versions = Object.keys(data).sort()
    
    let metadata = versions.map(v => {
      let res = data[v] as LrSnapshot;
      res.version = v;
      return res
    }).sort(
      (a,b) => a.version > b.version ? -1 : 1
    )
    
    let selected = metadata[0];

    setTimeout(() => {
      this.setState({
        metadata,
        selected
      })
    })    
  }
}

const SiteStructure = styled.div`
  display: flex;
`

const Nav = styled.div`
  padding: 18px;
  background: ${props => props.theme.colors.bgDarker};
  box-shadow: ${props => props.theme.shadows.default};
  margin-right: 12px;
  min-height: 100vh;
`

const NavItem = styled.div<{
  selected: boolean
}>`
  color: ${props => props.selected ? props.theme.colors.primary : "inherit"};
  cursor: pointer;
`

type VersionsProps = {
  versions: string[]
  selected: string
  onSelect: (string) => void
}
export class Versions extends React.Component<VersionsProps, {}> {
  render() {
    return (
      <Nav>
        {this.props.versions.map(v => (
          <NavItem key={v}
            selected={v == this.props.selected}
            onClick={() => this.props.onSelect(v)}
          >{v}</NavItem>
        ))}
      </Nav>
    )
  }
}

const Container = styled.div`
  padding: 24px;
`

const InfoLine = styled.div`
  padding: 0 0 12px 0;

  & span {
    margin-right: 48px;
    display: inline-block;
  }
`

type ResourcesProps = {
  snapshot: LrSnapshot
  prev: LrSnapshot
}
export class Resources extends React.Component<ResourcesProps, {}> {
  render() {
    let snap = this.props.snapshot
    window.SNAP = snap; // yeah yeah TODO: remove me

    let resources = snap.Resources.sort((a,b) => (a.ID > b.ID) ? 1 : -1 )
    // resources = this.computeUpdatedResources(resources, this.props.prev.Resources)

    let fieldsCnt = 0
    resources.forEach(r => {
      if (r.Body.Fields != null) {
        fieldsCnt += r.Body.Fields.length
      }
    })

    let cards = resources.map((resource, idx) => {
      return <Resource resource={resource} key={idx} />
    })
    
    return (
      <Container>
        <InfoLine>
          <span>Version: {snap.version}</span>
          <span>Resources: {snap.Resources.length}</span>
          <span>Fields: {fieldsCnt}</span>
        </InfoLine>
        <Cards>
          {cards}
        </Cards>
      </Container>
    )
  }

  computeUpdatedResources(resources: LrResource[], prevResources: LrResource[]): LrResource[] {
    let prev = {}
    prevResources.map(x => prev[x.ID] = x)

    let updatedResources = []
    resources.forEach(resource => {
      let prevResource = prev[resource.ID]
      if (prevResource == null) {
        resource.updated = true
        resource.Body.Fields.forEach(field => field.updated = true)
        updatedResources.push(resource)
        return
      }

      if (resource.Body.Fields == null) {
        return
      }

      let hasUpdates = false
      resource.Body.Fields.forEach((field) => {
        let updated = prevResource.Body.Fields.find(f => f.ID == field.ID) == null;
        if (updated) {
          field.updated = true
          hasUpdates = true
        }
      })

      if (hasUpdates) {
        updatedResources.push(resource)
        resource.updated = false
      }
    })

    return updatedResources
  }
}

const Cards = styled.div`
  display: flex;
  flex-wrap: wrap;
`;

const Card = styled.div`
  max-width: 400px;
  min-width: 200px;
  min-height: 100px;
  font-size: 24px;
  border: 2px solid #333;
  border-radius: 7px;
  margin: 0 24px 24px 0;
  padding: 15px 24px 24px 24px;
  background: ${props => props.theme.colors.bgDarker};

  &:hover {
    box-shadow: ${props => props.theme.shadows.default};
  }
`
const Name = styled.span<{
  updated?: boolean
}>`
  color: ${props => (props.updated !== false) ? props.theme.colors.primary : "white"};
`

const Inits = styled.span`
  font-size: 16px;
`

const FieldName = styled.span<{
  updated?: boolean
}>`
  color: ${props => (props.updated === true) ? props.theme.colors.primary : "inherit"};
  cursor: pointer;

  &:hover {
    color: ${props => props.theme.colors.secondary};
  }
`

const FieldType = styled.span`
  font-size: 16px;
  color: #aaa;
`;

type ResourceProps = {
  resource: LrResource
}
export class Resource extends React.Component<ResourceProps, {}> {
  render() {
    let { resource } = this.props

    let inits = this.renderInit(resource.Body.Inits)
    let listType = null;
    if (resource.ListType != null) {
      listType = "[]"+resource.ListType.Type.Type
    }

    let fields = (resource.Body.Fields || []).map((field) => this.renderField(field))

    return (
      <Card>
        <div>
          <Name updated={resource.updated}>{resource.ID}</Name> <Inits>{inits}</Inits>
        </div>
        {listType}
        {fields}
      </Card>
    )
  }

  renderInit(inits: LrInit[] | null): React.ReactElement | null {
    if (inits == null || inits.length == 0) {
      return (
        null
      )
    }

    if (inits.length != 1) {
      console.log("too many inits, ignoring everything else:")
      console.log(inits)
    }

    let init = inits[0];
    return (
      <span>
        ( {init.Args.map((arg) => arg.ID + " " + renderLrType(arg.Type)).join(", ")} )
      </span>
    )
  }

  renderField(field: LrField): React.ReactElement {
    return (
      <div key={field.ID}>
        <FieldName updated={field.updated}>{field.ID}</FieldName> <FieldType>{renderLrType(field.Type)}</FieldType>
      </div>
    )
  }
}

function renderLrType(t: LrType): string {
  if (t.SimpleType != null) return t.SimpleType.Type;
  if (t.ListType != null) return "[]"+renderLrType(t.ListType.Type);
  if (t.MapType != null) return "map["+t.MapType.Key.Type+"]"+renderLrType(t.MapType.Value);
  return "?";
}