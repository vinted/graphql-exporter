## GrapqhQL exporter

Exporter is designed to build and display metrics based on GraphQL query results.

`config_examle.json` shows basic usage case (in this case data from [Netbox](https://docs.netbox.dev)) and builds 3 metrics from PromQL query result given below:


```{
  "data": {
    "device_list": [
      {
        "name": "server",
        "serial": "1234",
        "custom_fields": {
          "price": "7408.80",
          "mgmt_mac": "11:11:11:11:11:00",
          "cpu_count": null,
          "mgmt_user": "root",
          "order_date": "2022-03-01",
          "memory_total": 512,
          "mgmt_password": "password",
          "price_per_month": "205.8",
          "depreciation_date": "2025-02-28",
          "depreciation_rate": "33.33",
          "order_contract_id": "contract-nr1",
          "storage_total_hdd": 0,
          "storage_total_ssd": 893
        }
      }
    ]
  }
}
```

Query supports dynamic `datetime` field. It can be specified as template variable.
For example when specified `{{NOW \"-1h\"}}` query will generate `datetime` that existed one hour ago.

Metrics results in:

```
# HELP graphql_exporter_custom_fields_depreciation_date Deprecation date
# TYPE graphql_exporter_custom_fields_depreciation_date gauge
graphql_exporter_custom_fields_depreciation_date{name="server",order_contract_id="contract-nr1",serial="1234",value="2025-02-28"} 1
# HELP graphql_exporter_custom_fields_memory_total Device memory total
# TYPE graphql_exporter_custom_fields_memory_total gauge
graphql_exporter_custom_fields_memory_total{name="server",order_contract_id="contract-nr1"} 512
# HELP graphql_exporter_custom_fields_price Device price
# TYPE graphql_exporter_custom_fields_price gauge
graphql_exporter_custom_fields_price{name="server",order_contract_id="contract-nr1"} 7408.8
```

API token can be overridden with `GRAPHQLAPITOKEN` env variable.
`CacheExpire` configuration parameter defines cache validity period. Value of `0` disables caching.
