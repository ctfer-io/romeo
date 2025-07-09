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
const os = __importStar(require("os"));
const adm_zip_1 = __importDefault(require("adm-zip"));
const child_process_1 = require("child_process");
const util_1 = require("util");
const execFileAsync = (0, util_1.promisify)(child_process_1.execFile);
function isPathInside(parent, child) {
    const relative = path.relative(parent, child);
    return (!!relative && !relative.startsWith('..') && !path.isAbsolute(relative));
}
async function unzipToDir(zipBuffer, targetDir) {
    const zip = new adm_zip_1.default(zipBuffer);
    const entries = zip.getEntries();
    for (const entry of entries) {
        const entryPath = path.join(targetDir, entry.entryName);
        if (!isPathInside(targetDir, entryPath)) {
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
}
async function run() {
    try {
        const url = core.getInput('server', { required: true });
        const strategy = core.getInput('strategy');
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
        const zipBuffer = Buffer.from(base64Zip, 'base64');
        if (strategy === 'raw') {
            const coverout = path.resolve('coverout');
            await fs.mkdir(coverout, { recursive: true });
            await unzipToDir(zipBuffer, coverout);
            core.setOutput('path', coverout);
        }
        else if (strategy === 'coverfile') {
            const coverfile = core.getInput('coverfile', { required: true });
            const tmpDir = await fs.mkdtemp(path.join(os.tmpdir(), 'cov-unzip-'));
            await unzipToDir(zipBuffer, tmpDir);
            await execFileAsync('go', [
                'tool',
                'covdata',
                'textfmt',
                `-i=${tmpDir}`,
                `-o=${coverfile}`
            ]);
            core.setOutput('path', coverfile);
        }
        else {
            core.setFailed(`Invalid mode "${strategy}". Must be either "coverfile" or "raw".`);
        }
    }
    catch (error) {
        core.setFailed(`${error?.message ?? error}`);
    }
}
void run();
//# sourceMappingURL=download.js.map