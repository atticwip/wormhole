const Wormhole = artifacts.require("Wormhole");
const WrappedAsset = artifacts.require("WrappedAsset");
const ERC20 = artifacts.require("ERC20PresetMinterPauser");

advanceBlock = () => {
    return new Promise((resolve, reject) => {
        web3.currentProvider.send({
            jsonrpc: "2.0",
            method: "eth_newBlockFilter",
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
        console.log("Bridge:", bridge.address);

        let token = await ERC20.deployed();
        console.log("Token:", token.address);

        while (true) {
            let ev = await bridge.lockAssets(
                /* asset address */
                token.address,
                /* amount */
                "1000000005",
                /* recipient */
                "0x00000000000000000000000090F8bf6A479f320ead074411a4B0e7944Ea8c9C1",
                /* target chain: ethereum */
                2,
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
