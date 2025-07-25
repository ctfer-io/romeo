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
var __importStar = (this && this.__importStar) || (function () {
    var ownKeys = function(o) {
        ownKeys = Object.getOwnPropertyNames || function (o) {
            var ar = [];
            for (var k in o) if (Object.prototype.hasOwnProperty.call(o, k)) ar[ar.length] = k;
            return ar;
        };
        return ownKeys(o);
    };
    return function (mod) {
        if (mod && mod.__esModule) return mod;
        var result = {};
        if (mod != null) for (var k = ownKeys(mod), i = 0; i < k.length; i++) if (k[i] !== "default") __createBinding(result, mod, k[i]);
        __setModuleDefault(result, mod);
        return result;
    };
})();
Object.defineProperty(exports, "__esModule", { value: true });
const core = __importStar(require("@actions/core"));
const stateHelper = __importStar(require("./state-helper"));
const fs = __importStar(require("./fs"));
const iac = __importStar(require("./iac"));
const dst = 'install';
async function run() {
    try {
        const stack = await iac.getStack(dst, dst);
        await stack.setAllConfig({
            'install:kubeconfig': {
                value: fs.resolveInput(core.getInput('kubeconfig')),
                secret: true
            },
            'install:namespace': {
                value: core.getInput('namespace')
            },
            'install:api-server': {
                value: core.getInput('api-server')
            },
            'install:harden': {
                value: core.getInput('harden', { required: false })
            }
        });
        const upRes = await stack.up({ onOutput: core.info });
        core.setOutput('kubeconfig', upRes.outputs['kubeconfig'].value);
        core.setOutput('namespace', upRes.outputs['namespace'].value);
    }
    catch (error) {
        core.setFailed(`${error?.message ?? error}`);
    }
}
async function cleanup() {
    try {
        const stack = await iac.getStack(dst, dst);
        await stack.destroy({ onOutput: core.info, remove: true });
    }
    catch (error) {
        core.warning(`${error?.message ?? error}`);
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
//# sourceMappingURL=install.js.map