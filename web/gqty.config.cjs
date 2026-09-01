/**
 * @type {import("@gqty/cli").GQtyConfig}
 */
const config = {
  react: true,
  scalarTypes: {
    Time: 'string',
  },
  introspection: {
    // Offline: local schema merged from graph/*.graphqls (see Makefile generate-web)
    endpoint: './schema.graphql',
  },
  destination: './src/gqty/index.ts',
  subscriptions: true,
};

module.exports = config;
