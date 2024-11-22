/// tries to send a transaction from a random wallet to another random address
/// expect to get a INSUFFICIENT_FUNDS error since nothing got filled up

const { Wallet, ethers } = require("ethers");

async function sendTransactionInsufficientFunds() {
    const provider = new ethers.JsonRpcProvider("http://127.0.0.1:8000");

    // Generate random wallets
    const wallet1 = Wallet.createRandom().connect(provider); // Connect wallet1 to the provider
    const wallet2 = Wallet.createRandom();

    const recipientAddress = wallet2.address;

    const tx = {
        to: recipientAddress,
        value: ethers.parseEther("0.01"),
        gasLimit: 21000,
        gasPrice: ethers.parseUnits("10", "gwei"),
    };

    try {
        const transactionResponse = await wallet1.sendTransaction(tx);
        console.log("Transaction sent! Hash:", transactionResponse.hash);

        const receipt = await transactionResponse.wait();
        console.log("Transaction mined! Receipt:", receipt);
    } catch (error) {
        console.error("Error sending transaction:", error);
    }
}

async function sendTransaction() {
    const provider = new ethers.JsonRpcProvider("http://127.0.0.1:8000");

    // Generate random wallets
    // omni devnet private keys can be found in lib/anvil/accounts.go
    const privKey1 = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80";
    const wallet1 = new ethers.Wallet(privKey1, provider);
    const wallet2 = Wallet.createRandom();

    const recipientAddress = wallet2.address;

    const tx = {
        to: recipientAddress,
        value: ethers.parseEther("0.01"),
        gasLimit: 21000,
        gasPrice: ethers.parseUnits("10", "gwei"),
    };

    try {
        const transactionResponse = await wallet1.sendTransaction(tx);
        console.log("Transaction sent! Hash:", transactionResponse.hash);

        const receipt = await transactionResponse.wait();
        console.log("Transaction mined! Receipt:", receipt);
    } catch (error) {
        console.error("Error sending transaction:", error);
    }
}

sendTransaction();
