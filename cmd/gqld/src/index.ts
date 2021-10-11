import express from "express"
import { postgraphile, PostGraphileOptions } from "postgraphile"
import { makeJSONPgSmartTagsPlugin, JSONPgSmartTags } from "graphile-utils"
import { PgNodeAliasPostGraphile } from "graphile-build-pg"
import tags from "./tags"

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
  appendPlugins: [tagsPlugin],
  skipPlugins: [PgNodeAliasPostGraphile],
  enhanceGraphiql: true,
  legacyRelations: "omit",

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
    "dealer",
    "packer",
    "piecer",
  ],
  options,
)

const app = express()

app.get("/", (_, res) => res.sendStatus(200))
app.use(p)

app.listen(process.env.PORT || 5000)
