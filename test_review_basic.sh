#!/bin/bash
# Basic test for review system functionality

echo "🧪 Testing Enhanced Review System"
echo "================================="

# 1. Check if test task exists in pending_review status
echo "1. Checking test task #999 exists in pending_review status..."
if grep -q '"id": 999' plan/task.json && grep -A1 '"id": 999' plan/task.json | grep -q '"status": "pending_review"'; then
    echo "   ✅ Test task #999 found in pending_review status"
else
    echo "   ❌ Test task #999 not found or not in pending_review status"
    exit 1
fi

# 2. Check if corresponding git branch exists
echo "2. Checking git branch task_999 exists..."
if git branch --list task_999 | grep -q task_999; then
    echo "   ✅ Branch task_999 exists"
else
    echo "   ❌ Branch task_999 not found"
    exit 1
fi

# 3. Check if app compiles successfully
echo "3. Testing application compilation..."
cd task-dashboard
if PATH=/usr/local/go/bin:$PATH go build -o test_build . >/dev/null 2>&1; then
    echo "   ✅ Application compiles successfully"
    rm -f test_build
else
    echo "   ❌ Application compilation failed"
    exit 1
fi
cd ..

# 4. Check UI bindings exist
echo "4. Checking TypeScript bindings for new functions..."
if grep -q "ApproveTask" task-dashboard/frontend/wailsjs/go/main/App.d.ts && grep -q "RejectTask" task-dashboard/frontend/wailsjs/go/main/App.d.ts; then
    echo "   ✅ ApproveTask and RejectTask bindings found"
else
    echo "   ❌ Missing function bindings"
    exit 1
fi

echo ""
echo "🎉 All basic tests passed!"
echo "📋 Review system ready for manual testing:"
echo "   • Navigate to http://localhost:34115"
echo "   • Look for test task #999 in Done column"
echo "   • Test Approve/Reject buttons"
echo ""