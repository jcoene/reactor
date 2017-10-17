import React from 'react';
import ReactDOMServer from 'react-dom/server';

const components = require.context('./', true, /\.jsx$/);

global.render = (req, cb) => {
  const component = components(`./${req.name}.jsx`)['default'];
  const html = ReactDOMServer.renderToString(React.createElement(component, req.props));
  const resp = {
    html: html,
  };
  cb(JSON.stringify(resp));
}
