import * as React from 'react';
import { render } from 'react-dom';
import styled, { ThemeProvider } from 'styled-components';
import 'whatwg-fetch';
import { LrDocs } from './src/lrdocs/lrdocs';

const theme = {
  colors: {
    primary: "#ff61a9",
    secondary: "#61fff9",
    fg: "#fff",
    bg: "#444",
    bgDarker: "#2b2b2b",
  },
  shadows: {
    default: "3px 3px 10px #111",
  },
};

const Background = styled.div`
  background: ${props => props.theme.colors.bg};
  color: ${props => props.theme.colors.fg};
  font-family: "Inconsolata", monospace;
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
`;

type AppState = {}
class App extends React.Component<{}, AppState> {
  state: AppState = {}

  render() {
    return (
      <ThemeProvider theme={theme}>
        {this.renderThemed.bind(this)()}
      </ThemeProvider>
    )
  }

  renderThemed() {
    return (
      <Background>
        <div style={{height: "100%", overflow: "auto"}}>
          <LrDocs />
        </div>
      </Background>
    )
  }
}

render(
  <App />,
  document.getElementById('root')
)