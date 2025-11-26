#!/usr/bin/env python3
"""
Script to validate that GRPC_DATA expected values in Keploy test reports are valid JSON.
This verifies that the protoscope to JSON conversion feature is working correctly.
"""

import json
import yaml
import sys
from pathlib import Path


def validate_grpc_data_json(report_file_path):
    """
    Validates that all GRPC_DATA expected values in the test report are valid JSON.
    
    Args:
        report_file_path: Path to the Keploy test report YAML file
    
    Returns:
        bool: True if all GRPC_DATA values are valid JSON, False otherwise
    """
    print("Checking protoscope to JSON feature - whether it's working correctly or not")
    print("=" * 80)
    
    # Load the YAML report file
    try:
        with open(report_file_path, 'r') as f:
            report = yaml.safe_load(f)
    except Exception as e:
        print(f"Error loading report file: {e}")
        return False
    
    # Get all tests from the report
    tests = report.get('tests', [])
    if not tests:
        print("No tests found in the report")
        return False
    
    print(f"Found {len(tests)} test(s) in the report\n")
    
    all_valid = True
    
    # Check each test
    for idx, test in enumerate(tests, 1):
        test_id = test.get('test_case_id', f'test-{idx}')
        print(f"Test {idx}: {test_id}")
        print("-" * 80)
        
        # Get body_result from the test result
        body_result = test.get('result', {}).get('body_result', [])
        
        # Find GRPC_DATA entries
        grpc_data_found = False
        for result_item in body_result:
            if result_item.get('type') == 'GRPC_DATA':
                grpc_data_found = True
                expected_value = result_item.get('expected', '')
                
                print(f"Checking the data type of the expected field of GRPC_DATA in test {idx}")
                print(f"Expected value type: {type(expected_value).__name__}")
                
                # Try to parse as JSON
                try:
                    parsed_json = json.loads(expected_value)
                    print(f"✓ Valid JSON found")
                    print(f"  JSON keys: {list(parsed_json.keys())}")
                    print()
                except json.JSONDecodeError as e:
                    print(f"✗ Invalid JSON - Parse error: {e}")
                    print(f"  Value preview: {expected_value[:100]}...")
                    print()
                    all_valid = False
        
        if not grpc_data_found:
            print(f"⚠ No GRPC_DATA found in test {idx}")
            print()
    
    print("=" * 80)
    if all_valid:
        print("✓ Found valid JSON, feature is working")
        return True
    else:
        print("✗ Some GRPC_DATA values are not valid JSON")
        return False


if __name__ == "__main__":
    # Default to the test report file in the workspace
    default_report = Path(__file__).parent / "keploy/reports/test-run-1/test-set-0-report.yaml"
    
    if len(sys.argv) > 1:
        report_path = sys.argv[1]
    else:
        report_path = default_report
    
    if not Path(report_path).exists():
        print(f"Error: Report file not found: {report_path}")
        sys.exit(1)
    
    success = validate_grpc_data_json(report_path)
    sys.exit(0 if success else 1)
