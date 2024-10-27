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
const buffer_1 = require("buffer");
const path = __importStar(require("path"));
const fs = __importStar(require("fs"));
const yauzl = __importStar(require("yauzl"));
void import('node-fetch');
async function run() {
    try {
        const server = core.getInput('server');
        const response = await fetch(`${server}/coverout`);
        const data = (await response.json());
        await extract(data.merged, core.getInput('directory'));
    }
    catch (error) {
        core.setFailed(`${error?.message ?? error}`);
    }
}
async function extract(merged, outputDir) {
    // 1. Decode base64
    const decoded = buffer_1.Buffer.from(merged, 'base64');
    // 2. Unzip content
    yauzl.fromBuffer(decoded, { lazyEntries: true }, (err, zipFile) => {
        if (err)
            throw err;
        if (!fs.existsSync(outputDir)) {
            fs.mkdirSync(outputDir, { recursive: true });
        }
        zipFile.readEntry();
        zipFile.on('entry', (entry) => {
            const filePath = path.join(outputDir, entry.fileName);
            if (entry.fileName.endsWith('/')) {
                // Create directory if not exist
                fs.mkdirSync(filePath, { recursive: true });
                zipFile.readEntry();
            }
            else {
                // Extract and save file
                zipFile.openReadStream(entry, (suberr, readStream) => {
                    if (suberr)
                        throw suberr;
                    fs.mkdirSync(path.dirname(filePath), { recursive: true });
                    const writeStream = fs.createWriteStream(filePath);
                    readStream.pipe(writeStream);
                    readStream.on('end', () => zipFile.readEntry());
                });
            }
        });
    });
}
// Main
void run();
//# sourceMappingURL=download.js.map