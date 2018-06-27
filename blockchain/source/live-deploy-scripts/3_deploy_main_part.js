var multiSigMigrations = artifacts.require('./MultiSigWallet.sol');
var multiSigOracle = artifacts.require('./MultiSigWallet.sol');
var Market = artifacts.require('./Market.sol');
var SNM = artifacts.require('./SNM.sol');
var Blacklist = artifacts.require('./Blacklist.sol');
var ProfileRegistry = artifacts.require('./ProfileRegistry');
var Oracle = artifacts.require('./OracleUSD.sol');
var GateKeeper = artifacts.require('./SimpleGatekeeperWithLimit.sol');
var DeployList = artifacts.require('./DeployList.sol');
var AddressHashMap = artifacts.require('./AddressHashMap.sol');

// filled before deploy
var SNMMasterchainAddress = '0x983f6d60db79ea8ca4eb9968c6aff8cfa04b3c63';

ar MSOwners = ['0xdaec8F2cDf27aD3DF5438E5244aE206c5FcF7fCd', '0xd9a43e16e78c86cf7b525c305f8e72723e0fab5e', '0x72cb2a9AD34aa126fC02b7d32413725A1B478888', '0x1f50Be5cbFBFBF3aBD889e17cb77D31dA2Bd7227', '0xe062C67207F7E478a93EF9BEA39535d8EfFAE3cE', '0x5fa359a9137cc5ac2a85d701ce2991cab5dcd538', '0x7aa5237e0f999a9853a9cc8c56093220142ce48e', '0xd43f262536e916a4a807d27080092f190e25d774', '0xdd8422eed7fe5f85ea8058d273d3f5c17ef41d1c'];
var MSRequired = 5;

var benchmarksQuantity = 13;
var netflagsQuantity = 3;

var Deployers = ['0x7aa5237e0f999a9853a9cc8c56093220142ce48e', '0xd9a43e16e78c86cf7b525c305f8e72723e0fab5e'];

// main part

module.exports = function (deployer, network) {
    deployer.then(async () => {
	    if (network === 'privateLive') {
	    	let gk = await GateKeeper.deployed();
	        await deployer.deploy(multiSigMigrations, MSOwners, MSRequired, { gasPrice: 0 });
	        let multiSigMig = await multiSigMigrations.deployed();
	        await deployer.deploy(ProfileRegistry, { gasPrice: 0 });
	        let pr = await ProfileRegistry.deployed();
	        await pr.transferOwnership(multiSigMig.address, { gasPrice: 0 });
	    	await deployer.deploy(Blacklist, { gasPrice: 0 });
	        let bl = await Blacklist.deployed();
	        await deployer.deploy(multiSigOracle, MSOwners, MSRequired, { gasPrice: 0 });
	    	let multiSigOrac = await multiSigOracle.deployed();
	        await deployer.deploy(Oracle, { gasPrice: 0 });
	        let oracle = await Oracle.deployed();
	       	oracle.transferOwnership(multiSigOrac.address, { gasPrice: 0 });      
	        await deployer.deploy(Market, SNM.address, Blacklist.address, Oracle.address, ProfileRegistry.address, benchmarksQuantity, netflagsQuantity, { gasPrice: 0 });
	        let market = await Market.deployed();
	        await market.transferOwnership(multiSigMigrations.address, { gasPrice: 0 });
	        await bl.SetMarketAddress(market.address, { gasPrice: 0 });
	  		await bl.transferOwnership(multiSigMig.address, { gasPrice: 0 });
	  		await deployer.deploy(DeployList, Deployers, { gasPrice: 0 })
	  		await deployer.deploy(AddressHashMap, { gasPrice: 0 });
	  		let dl = await DeployList.deployed();
	  		let ahm = await AddressHashMap.deployed();
	  		await ahm.write('sidechainSNMAddress', SNM.address, { gasPrice: 0 });
	  		await ahm.write('masterchainSNMAddress', SNMMasterchainAddress, { gasPrice: 0 });
	  		await ahm.write('blacklistAddress', bl.address, { gasPrice: 0 });
	  		await ahm.write('marketAddress', market.address, { gasPrice: 0 });
	  		await ahm.write('profileRegistryAddress', pr.address, { gasPrice: 0 });
	  		await ahm.write('oracleUsdAddress', oracle.address, { gasPrice: 0 });
	  		await ahm.write('gatekeeperSidechainAddress', gk.address, { gasPrice: 0 });
	  		await dl.transferOwnership(multiSigMig.address, { gasPrice: 0 });
	  		await ahm.transferOwnership(multiSigMig.address, { gasPrice: 0 });
	    }
    });
};
