import { makeExtendSchemaPlugin, gql, embed } from "graphile-utils"

const MyPlugin = makeExtendSchemaPlugin(build => {
  // Get any helpers we need from `build`
  const { pgSql: sql, inflection, graphql } = build

  return {
    typeDefs: gql`
      extend type Batch {
        xxx: Int @pgQuery(
          fragment: ${embed(sql.fragment`(SELECT floor(random() * 10 + 1)::int)`)}
        )
        yyy: String @pgQuery(
          fragment: ${embed(
            (queryBuilder: any) => sql.fragment`(select batch_id from packer.batches where batch_id = ${queryBuilder.getTableAlias()}.id)`
          )}
        )
        zzz: String @pgQuery(
          fragment: ${embed(
            (queryBuilder: any) => sql.fragment`(select origin from packer.batches where batch_id = ${queryBuilder.getTableAlias()}.id)`
          )}
        )
      }
    `,
    resolvers: {
      /*...*/
    },
  }
})

export default MyPlugin