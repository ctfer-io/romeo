"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
Object.defineProperty(exports, "__esModule", { value: true });
const core = __importStar(require("@actions/core"));
const stateHelper = __importStar(require("./state-helper"));
const iac = __importStar(require("./iac"));
async function run() {
    try {
        const stack = await iac.getStack('environment');
        await stack.setConfig('romeo-environment:kubeconfig', {
            value: core.getInput('kubeconfig')
        });
        console.info('Deploying Romeo environment...');
        const upRes = await stack.up({ onOutput: console.info });
        core.setOutput('port', upRes.outputs.port.value);
        core.setOutput('claim-name', upRes.outputs.claimName.value);
    }
    catch (error) {
        core.setFailed(`Action failed: ${error?.message ?? error}`);
    }
}
async function cleanup() {
    try {
        const stack = await iac.getStack('environment');
        await stack.destroy({ onOutput: console.info, remove: true });
        core.info('Romeo environment destroyed.');
    }
    catch (error) {
        core.warning(`Cleanup failed: ${error?.message ?? error}`);
    }
}
// Main
if (!stateHelper.IsPost) {
    void run();
}
// Post
else {
    void cleanup();
}
//# sourceMappingURL=environment.js.map