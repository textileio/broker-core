import { createServer } from 'http';
import { postgraphile, PostGraphileOptions } from 'postgraphile'
import { makeJSONPgSmartTagsPlugin, JSONPgSmartTags } from 'graphile-utils'
import { PgNodeAliasPostGraphile } from 'graphile-build-pg'
import tags from './tags'
import plugin from './plugin'

const tagsPlugin = makeJSONPgSmartTagsPlugin(tags)

let options: PostGraphileOptions = {
  enableCors: true,
    dynamicJson: true,
    setofFunctionsContainNulls: false,
    disableDefaultMutations: true,
    ignoreRBAC: false,
    ignoreIndexes: true,
    watchPg: false,
    graphiql: true,
    appendPlugins: [tagsPlugin, plugin],
    skipPlugins: [PgNodeAliasPostGraphile],
    enhanceGraphiql: true,
    
    // Will be changed in DEV mode
    showErrorStack: false,
    extendedErrors: undefined,
    disableQueryLog: true,
    allowExplain: false,
}

if (process.env.DEV) {
  options.showErrorStack = "json"
  options.extendedErrors = ["hint"]
  options.disableQueryLog = false
  options.allowExplain = true
}

let p = postgraphile(
  process.env.DATABASE_URL,
  [
    "auctioneer",
    "broker",
    "packer",
  ],
  options,
)

createServer(p).listen(process.env.PORT || 5000)