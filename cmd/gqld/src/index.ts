import { createServer } from 'http';
import { postgraphile, PostGraphileOptions } from 'postgraphile'
import { makeJSONPgSmartTagsPlugin, JSONPgSmartTags } from 'graphile-utils'
import tags from './tags'

const plugin = makeJSONPgSmartTagsPlugin(tags)

let options: PostGraphileOptions = {
  enableCors: true,
    dynamicJson: true,
    setofFunctionsContainNulls: false,
    disableDefaultMutations: true,
    ignoreRBAC: false,
    ignoreIndexes: false,
    watchPg: false,
    graphiql: true,
    appendPlugins: [plugin],
    
    // Will be changed in DEV mode
    showErrorStack: false,
    extendedErrors: undefined,
    enhanceGraphiql: false,
    disableQueryLog: true,
    allowExplain: false,
}

if (process.env.DEV) {
  options.showErrorStack = "json"
  options.extendedErrors = ["hint"]
  options.enhanceGraphiql = true,
  options.disableQueryLog = false
  options.allowExplain = true
}

let p = postgraphile(
  process.env.DATABASE_URL,
  [
    "auctioneer",
    "broker",
  ],
  options,
)

createServer(p).listen(process.env.PORT || 5000)