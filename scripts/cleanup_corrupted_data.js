// MongoDB script to identify and optionally delete corrupted mock API records
// Run this in MongoDB shell: mongosh < cleanup_corrupted_data.js

// Connect to your database
use mocktool;

print("=== Checking for corrupted BSON records ===\n");

// Find all mock APIs
const allApis = db.mock_apis.find({}).toArray();
print(`Total mock APIs found: ${allApis.length}\n`);

const corruptedRecords = [];

allApis.forEach((api, index) => {
    let hasIssue = false;
    let issues = [];

    // Check hash_input
    if (api.hash_input) {
        try {
            // Try to access the field - if it's corrupted, this might fail
            const hashInput = api.hash_input;
            if (hashInput && typeof hashInput === 'object') {
                // Try to stringify to verify it's valid
                JSON.stringify(hashInput);
            }
        } catch (e) {
            hasIssue = true;
            issues.push('hash_input: ' + e.message);
        }
    }

    // Check output
    if (api.output) {
        try {
            const output = api.output;
            if (output && typeof output === 'object') {
                JSON.stringify(output);
            }
        } catch (e) {
            hasIssue = true;
            issues.push('output: ' + e.message);
        }
    }

    if (hasIssue) {
        corruptedRecords.push({
            _id: api._id,
            name: api.name,
            feature_name: api.feature_name,
            scenario_name: api.scenario_name,
            issues: issues
        });
    }
});

print(`\n=== Found ${corruptedRecords.length} corrupted records ===\n`);

if (corruptedRecords.length > 0) {
    corruptedRecords.forEach((record, i) => {
        print(`${i + 1}. ID: ${record._id}`);
        print(`   Name: ${record.name}`);
        print(`   Feature: ${record.feature_name}`);
        print(`   Scenario: ${record.scenario_name}`);
        print(`   Issues: ${record.issues.join(', ')}`);
        print('');
    });

    print("\n=== To delete these corrupted records, uncomment and run: ===");
    print("// db.mock_apis.deleteMany({ _id: { $in: [");
    corruptedRecords.forEach((record, i) => {
        const comma = i < corruptedRecords.length - 1 ? ',' : '';
        print(`//   ObjectId("${record._id}")${comma}`);
    });
    print("// ] } });");

    print("\n=== Or delete ALL mock APIs and start fresh: ===");
    print("// db.mock_apis.deleteMany({});");
} else {
    print("No obviously corrupted records found via JavaScript checks.");
    print("The corruption may be at the BSON byte level.");
    print("\nTo view the specific corrupted record mentioned in the error:");
    print('db.mock_apis.findOne({ _id: ObjectId("694566766cf626df43e7a853") });');
    print("\nTo delete it:");
    print('db.mock_apis.deleteOne({ _id: ObjectId("694566766cf626df43e7a853") });');
}
