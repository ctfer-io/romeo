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
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const core = __importStar(require("@actions/core"));
const node_fetch_1 = __importDefault(require("node-fetch"));
const fs = __importStar(require("fs/promises"));
const path = __importStar(require("path"));
const adm_zip_1 = __importDefault(require("adm-zip"));
function isPathInside(parent, child) {
    const relative = path.relative(parent, child);
    return (!!relative && !relative.startsWith('..') && !path.isAbsolute(relative));
}
async function run() {
    try {
        const url = core.getInput('server', { required: true });
        const directory = core.getInput('directory', { required: true });
        await fs.mkdir(directory, { recursive: true });
        // Fetch JSON
        const response = await (0, node_fetch_1.default)(`${url}/api/v1/coverout`);
        if (!response.ok) {
            throw new Error(`Failed to fetch: ${response.statusText}`);
        }
        const json = (await response.json());
        const base64Zip = json.merged;
        if (!base64Zip || typeof base64Zip !== 'string') {
            throw new Error('Invalid or missing "merged" attribute in JSON.');
        }
        // Decode Base64
        const zipBuffer = Buffer.from(base64Zip, 'base64');
        // Unzip securely
        const zip = new adm_zip_1.default(zipBuffer);
        const entries = zip.getEntries();
        for (const entry of entries) {
            const entryPath = path.join(directory, entry.entryName);
            if (!isPathInside(directory, entryPath)) {
                throw new Error(`Zip traversal attack detected: ${entry.entryName}`);
            }
            if (entry.isDirectory) {
                await fs.mkdir(entryPath, { recursive: true });
            }
            else {
                await fs.mkdir(path.dirname(entryPath), { recursive: true });
                await fs.writeFile(entryPath, entry.getData());
            }
        }
        // Set output
        core.setOutput('directory', directory);
    }
    catch (error) {
        core.setFailed(error.message);
    }
}
run();
//# sourceMappingURL=download.js.map