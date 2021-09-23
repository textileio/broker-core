import { JSONPgSmartTags } from 'graphile-utils'

const tags: JSONPgSmartTags = {
  version: 1,
  config: {
    class: {
      /*
      * API
      */
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
          foreignKey: "(batch_id) references broker.batches (id)",
        },
        attribute: {
          "batch_id": {
            tags: {
              name: "storagePayloadId",
            }
          }
        }
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
      "broker.batches": {
        tags: {
          name: "storagePayloads"
        }
      },
      "broker.deals": {
        attribute: {
          "batch_id": {
            tags: {
              name: "storagePayloadId",
            }
          }
        }
      },
      "broker.storage_requests": {
        attribute: {
          "batch_id": {
            tags: {
              name: "storagePayloadId",
            }
          }
        }
      },
      "broker.batch_manifests": {
        tags: {
          name: "storagePayloadManifests",
          foreignKey: "(batch_id) references broker.batches (id)",
        },
        attribute: {
          "batch_id": {
            tags: {
              name: "storagePayloadId",
            }
          }
        }
      },
      "broker.batch_tags": {
        tags: {
          name: "storagePayloadTags",
        },
        attribute: {
          "batch_id": {
            tags: {
              name: "storagePayloadId",
            }
          }
        }
      },
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
      "broker.batch_remote_wallet": {
        tags: {
          omit: "create,read,update,delete,filter,order,all,many,execute",
        },
      },




      /*
      * Dealer
      */
      "dealer.schema_migrations": {
        tags: {
          omit: "create,read,update,delete,filter,order,all,many,execute",
        },
      },



      /*
      * Packer
      */
      "packer.batches": {
        tags: {
          foreignKey: "(batch_id) references broker.batches (id)",
        },
        attribute: {
          "batch_id": {
            tags: {
              name: "storagePayloadId",
            }
          },
        }
      },
      "packer.schema_migrations": {
        tags: {
          omit: "create,read,update,delete,filter,order,all,many,execute",
        },
      },
      "packer.storage_requests": {
        tags: {
          name: "packer.storage_requests.ignored",
          omit: "create,read,update,delete,filter,order,all,many,execute",
        },
      },



      /*
      * Piecer
      */
      "piecer.schema_migrations": {
        tags: {
          omit: "create,read,update,delete,filter,order,all,many,execute",
        },
      },
    },
  },
}

export default tags