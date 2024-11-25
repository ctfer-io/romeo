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
exports.getStack = getStack;
const automation_1 = require("@pulumi/pulumi/automation");
const upath = __importStar(require("upath"));
async function getStack(dst) {
    // Create our stack using a local program
    // in the ../deploy directory
    const args = {
        stackName: 'romeo',
        // All Romeo actions IaC are contained in <action>/deploy so we only need the action name
        workDir: upath.joinSafe(__dirname, '..', dst, 'deploy')
    };
    const opts = {
        envVars: {
            PULUMI_CONFIG_PASSPHRASE: ''
        }
    };
    // create (or select if one already exists) a stack that uses our local program
    return automation_1.LocalWorkspace.createOrSelectStack(args, opts);
}
//# sourceMappingURL=iac.js.map