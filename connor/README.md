<p align="center">
  <h3 align="center">Arbitrage Bot</h3>
</p>

---

### Description
Arbitrage bot estimates the mining profitability of a certain cryptocurrency within a specified time interval.
Based on the decision, the bot selects the cryptocurrency and creates orders for buying capacities on the market.
Bot checks the information about workers on the market and makes further decisions on the purchase capacity.

### Tasks
1. Get profitability of crypto-currency for calculate mining profit;
2. Get condition from SONM market. If the market is empty, a lot of orders are placed on the market at the maximum settlement price;
3. Get Active deals from DWH for reinvoice order and deploy a new container for pool;
4. Get rewards from pool => exchange on wallets

### How to work
Use <bot.yaml>: get pool addresses, eth addresses and tickers info.

-------------------------------------------
| Token      | BenchMap                                         |     Step | BuyHashRate|
| -----------|:------------------------------------------------:|:--------:|:----------:|
| ETH     | "gpu-eth-hashrate": hashrate * (buyHashrate + step) | 0.01 Mh/s |10-180 Mh/s
| ZEC     | "cash-hashrate-gpu": buyHashrate + step             |  1 Sol/s  |2050 - 2070 Sol/s
| XMR     | "gpu-eth-hashrate": buyHashrate + step              |    1 H/s  |3240 H/s