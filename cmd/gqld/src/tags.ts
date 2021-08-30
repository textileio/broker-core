import { JSONPgSmartTags } from 'graphile-utils'

const tags: JSONPgSmartTags = {
  version: 1,
  config: {
    class: {
      "api.schema_migrations": {
        tags: {
          omit: "create,read,update,delete,filter,order,all,many,execute",
        },
      },
      "api.auctions": {
        tags: {
          name: "apiAuctions",
        },
      },
      "api.bids": {
        tags: {
          name: "apiBids",
        },
      },


      /*
      * Auctioneer
      */
      "auctioneer.auctions": {
        tags: {
          foreignKey: "(batch_id) references broker.batches (id)"
        },
      },
      "auctioneer.bids": {
        attribute: {
          wallet_addr_sig: {
            tags: {
              omit: "create,read,update,delete,filter,order,all,many,execute",
            },
          },
        },
      },
      "auctioneer.schema_migrations": {
        tags: {
          omit: "create,read,update,delete,filter,order,all,many,execute",
        },
      },

      /*
      * Broker
      */
      "broker.operations": {
        tags: {
          omit: "create,read,update,delete,filter,order,all,many,execute",
        },
      },
      "broker.schema_migrations": {
        tags: {
          omit: "create,read,update,delete,filter,order,all,many,execute",
        },
      },
      "broker.unpin_jobs": {
        tags: {
          omit: "create,read,update,delete,filter,order,all,many,execute",
        },
      },


      "dealer.schema_migrations": {
        tags: {
          omit: "create,read,update,delete,filter,order,all,many,execute",
        },
      },
      "packer.schema_migrations": {
        tags: {
          omit: "create,read,update,delete,filter,order,all,many,execute",
        },
      },
      "piecer.schema_migrations": {
        tags: {
          omit: "create,read,update,delete,filter,order,all,many,execute",
        },
      },
    },
  },
}

export default tags