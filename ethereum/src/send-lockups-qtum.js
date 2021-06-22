const Wormhole = artifacts.require("Wormhole");
const WrappedAsset = artifacts.require("WrappedAsset");
const ERC20 = artifacts.require("ERC20PresetMinterPauser");

advanceBlock = () => {
    return new Promise((resolve, reject) => {
        web3.currentProvider.send({
            jsonrpc: "2.0",
            method: "evm_mine",
            id: new Date().getTime()
        }, (err, result) => {
            if (err) {
                return reject(err);
            }
            const newBlockHash = web3.eth.getBlock('latest').hash;

            return resolve(newBlockHash)
        });
    });
}

function sleep(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
}

module.exports = function(callback) {
    const fn = async () => {
        let bridge = await Wormhole.deployed();
        let token = await ERC20.deployed();
        console.log("Token:", token.address);

        while (true) {
            let ev = await bridge.lockAssets(
                /* asset address */
                token.address,
                /* amount */
                "1000000005",
                /* recipient */
                "0x0000000000000000000000007926223070547d2d15b2ef5e7383e541c338ffe9",
                /* target chain: qtum */
                4,
                /* nonce */
                Math.floor(Math.random() * 65535),
                /* refund dust? */
                false
            );

            let block = await web3.eth.getBlock('latest');
            console.log("block", block.number, "with txs", block.transactions, "and time", block.timestamp);
            await advanceBlock();
            await sleep(5000);
        }
    }

    fn().catch(reason => console.error(reason))
}
