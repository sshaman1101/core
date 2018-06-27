var sidechainGateMS = artifacts.require('./MultiSigWallet.sol');
var GateKeeper = artifacts.require('./SimpleGatekeeperWithLimit.sol');
var SNM = artifacts.require('./SNM.sol');

ar MSOwners = ['0xdaec8F2cDf27aD3DF5438E5244aE206c5FcF7fCd', '0xd9a43e16e78c86cf7b525c305f8e72723e0fab5e', '0x72cb2a9AD34aa126fC02b7d32413725A1B478888', '0x1f50Be5cbFBFBF3aBD889e17cb77D31dA2Bd7227', '0xe062C67207F7E478a93EF9BEA39535d8EfFAE3cE', '0x5fa359a9137cc5ac2a85d701ce2991cab5dcd538', '0x7aa5237e0f999a9853a9cc8c56093220142ce48e', '0xd43f262536e916a4a807d27080092f190e25d774', '0xdd8422eed7fe5f85ea8058d273d3f5c17ef41d1c'];
var MSRequired = 5;
var MSRequired = 1;
var freezingTime = 60 *15;

module.exports = function (deployer, network) {
    deployer.then(async () => {
	    if (network === 'privateLive') {
	    	await deployer.deploy(SNM, { gasPrice: 0 });
	    	let token = await SNM.deployed();
	        await deployer.deploy(sidechainGateMS, MSOwners, MSRequired, { gasPrice: 0 });
	        let multiSig = await sidechainGateMS.deployed();
	        await token.transfer(multiSig.address, 444 * 1e6 * 1e18, { gasPrice: 0 });
	        await deployer.deploy(GateKeeper, SNM.address, freezingTime, { gasPrice: 0 });
	        let Gate = await GateKeeper.deployed();
	        await Gate.transferOwnership(multiSig.address, { gasPrice: 0 });
	    }
    });
};
