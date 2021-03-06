import { JSONPgSmartTags } from 'graphile-utils'

const tags: JSONPgSmartTags = {
  version: 1,
  config: {
    class: {
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
        tags: {
          foreignKey: "(storage_provider_id) references auctioneer.storage_providers (id)",
        },
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
      "broker.storage_requests": {
        tags: {
          foreignKey: "(batch_id) references packer.batches (batch_id)",
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
      "broker.deals": {
        tags: {
          omit: "create,read,update,delete,filter,order,all,many,execute",
          name: "breoker.deals.ignored",
          foreignKey: [
            "(auction_id) references auctioneer.auctions (id)",
            "(bid_id) references auctioneer.bids (id)",
          ],
          primaryKey: "storage_provider_id,auction_id",
          unique: ["bid_id,auction_id", "storage_provider_id,bid_id", "bid_id"]
        },
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
          omit: "create,read,update,delete,filter,order,all,many,execute",
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
      "dealer.auction_deals": {
        tags: {
          name: "deals",
          foreignKey: [
            "(batch_id) references broker.batches (id)",
            "(auction_id) references auctioneer.auctions (id)",
            "(bid_id) references auctioneer.bids (id)",
            "(storage_provider_id) references auctioneer.storage_providers (id)",
          ],
          primaryKey: "storage_provider_id,auction_id",
          unique: ["bid_id,auction_id", "storage_provider_id,bid_id", "bid_id"]
        },
        attribute: {
          "batch_id": {
            tags: {
              name: "storagePayloadId",
            }
          },
          "id": {
            tags: {
              omit: "create,read,update,delete,filter,order,all,many,execute",
            }
          },
          "auction_data_id": {
            tags: {
              omit: "create,read,update,delete,filter,order,all,many,execute",
            }
          }
        }
      },
      "dealer.auction_data": {
        tags: {
          omit: "create,read,update,delete,filter,order,all,many,execute",
        },
      },
      "dealer.remote_wallet": {
        tags: {
          omit: "create,read,update,delete,filter,order,all,many,execute",
        },
      },
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
      "packer.storage_requests": {
        tags: {
          name: "packer.storage_requests.ignored",
          omit: "create,read,update,delete,filter,order,all,many,execute",
        },
      },
      "packer.schema_migrations": {
        tags: {
          omit: "create,read,update,delete,filter,order,all,many,execute",
        },
      },



      /*
      * Piecer
      */
      "piecer.unprepared_batches": {
        tags: {
          name: "pieceInfo",
          foreignKey: "(batch_id) references packer.batches (batch_id)",
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
