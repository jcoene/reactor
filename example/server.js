import React from 'react';
import ReactDOMServer from 'react-dom/server';

const components = require.context('./', true, /\.jsx$/);

global.render = (json, cb) => {
  const req = JSON.parse(json);
  const component = components(`./${req.name}.jsx`)['default'];
  const html = ReactDOMServer.renderToString(React.createElement(component, req.props));
  return JSON.stringify({html: html});
}
