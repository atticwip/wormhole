const WrappedAsset = artifacts.require("WrappedAsset");
const Wormhole = artifacts.require("Wormhole");
const ERC20 = artifacts.require("ERC20PresetMinterPauser");

module.exports = async function (deployer) {
    let bridge = await Wormhole.deployed();
    let token = await ERC20.deployed();

    console.log("Token:", token.address);
    console.log("Wormhole:", bridge.address);

    // Create example ERC20 and mint a generous amount of it.
    await token.mint("0x7926223070547d2d15b2ef5e7383e541c338ffe9", "1000000000000000000");

    await token.approve(bridge.address, "1000000000000000000");
};
