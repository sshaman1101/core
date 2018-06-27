var GateMultisig = artifacts.require('./MultiSigWallet.sol');
var GateKeeperLive = artifacts.require('./SimpleGatekeeperWithLimitLive.sol');

var MSOwners = ['0xdaec8F2cDf27aD3DF5438E5244aE206c5FcF7fCd', '0xd9a43e16e78c86cf7b525c305f8e72723e0fab5e', '0x72cb2a9AD34aa126fC02b7d32413725A1B478888', '0x1f50Be5cbFBFBF3aBD889e17cb77D31dA2Bd7227', '0xe062C67207F7E478a93EF9BEA39535d8EfFAE3cE', '0x5fa359a9137cc5ac2a85d701ce2991cab5dcd538', '0x7aa5237e0f999a9853a9cc8c56093220142ce48e', '0xd43f262536e916a4a807d27080092f190e25d774', '0xdd8422eed7fe5f85ea8058d273d3f5c17ef41d1c'];
var MSRequired = 5;
var freezingTime = 60 * 15;
var SNMMasterchainAddress = '0x983f6d60db79ea8ca4eb9968c6aff8cfa04b3c63';
var actualGasPrice = 400;

module.exports = function (deployer, network) {
    deployer.then(async () => {
	    if (network === 'master') {
	        await deployer.deploy(GateMultisig, MSOwners, MSRequired, { gasPrice: actualGasPrice });
	        await deployer.deploy(GateKeeperLive, SNMMasterchainAddress, freezingTime, { gasPrice: actualGasPrice });
	        let multisig = await GateMultisig.deployed();
	        let gk = await GateKeeperLive.deployed();
	        await gk.tranferOwnership(multisig.address);
	    }
    });
};
