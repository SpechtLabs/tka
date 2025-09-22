---
title: Configure TKA capability in Tailscale ACLs
permalink: /how-to/configure-acl
createTime: 2025/01/27 10:00:00
---

This guide shows you how to add capability grants to your Tailscale ACL policy, mapping users and groups to Kubernetes roles with validity periods.

## Understanding Capability Grants

TKA uses Tailscale's [capability grants](https://tailscale.com/kb/1324/grants) to determine:

- **Who** can access TKA (users, groups, devices)
- **What role** they get in Kubernetes
- **How long** their access lasts

## Basic Setup

:::: steps

1. ### Tag Your TKA Servers

   First, define who can manage TKA servers:

   ```jsonc
   {
     "tagOwners": {
       "tag:tka": ["group:admins", "alice@example.com"]
     }
   }
   ```

   **`tag:tka`**: Tag applied to TKA server devices
   **`tagOwners`**: Who can create/manage TKA-tagged devices

2. ### Create Capability Grants

   Add grants that map users/groups to Kubernetes roles:

   ```jsonc
   {
     "grants": [
       {
         "src": ["group:admins"],           // Who gets access
         "dst": ["tag:tka"],                // TKA servers
         "ip": ["443"],                     // Port (must match server)
         "app": {
           "specht-labs.de/cap/tka": [      // Capability name
             {
               "role": "cluster-admin",     // Kubernetes role
               "period": "8h",              // Token lifetime
               "priority": 100              // Priority - higher priority takes precedence over lower priority
             }
           ]
         }
       }
     ]
   }
   ```

::::

## Multiple Roles Example

Configure different access levels for different groups:

```jsonc
{
  "groups": {
    "group:k8s-admins": ["alice@example.com", "bob@example.com"],
    "group:developers": ["charlie@example.com", "diana@example.com"],
    "group:viewers": ["eve@example.com"]
  },
  "tagOwners": {
    "tag:tka": ["group:k8s-admins"]
  },
  "grants": [
    // Most engineers will get viewing permissions in k8s to see what's running
    {
      "src": ["group:k8s-viewers"],
      "dst": ["tag:tka"],
      "ip": ["443"],
      "app": {
        "specht-labs.de/cap/tka": [
          {
            "role": "view",
            "period": "4h",
            "priority": 100
          }
        ]
      }
    },
    // Some developers get limited write access
    {
      "src": ["group:developers"],
      "dst": ["tag:tka"],
      "ip": ["443"],
      "app": {
        "specht-labs.de/cap/tka": [
          {
            "role": "limited-edit",
            "period": "4h",
            "priority": 101
          }
        ]
      }
    },
    // OnCall Engineers get admin access during their on-call shift
    // This group membership is managed using Just-in-time access (https://tailscale.com/kb/1443/)
    {
      "src": ["group:developer-oncall"],
      "dst": ["tag:tka"],
      "ip": ["443"],
      "app": {
        "specht-labs.de/cap/tka": [
          {
            "role": "admin",
            "period": "1h",
            "priority": 200
          }
        ]
      }
    },
    // Platform OnCall Engineers get admin access during their on-call shift
    // This group membership is managed using Just-in-time access (https://tailscale.com/kb/1443/)
    {
      "src": ["group:platform-oncall"],
      "dst": ["tag:tka"],
      "ip": ["443"],
      "app": {
        "specht-labs.de/cap/tka": [
          {
            "role": "cluster-admin",
            "period": "1h",
            "priority": 300
          }
        ]
      }
    },
    // Full admin access for 8 hours only for our k8s-admins
    {
      "src": ["group:k8s-admins"],
      "dst": ["tag:tka"],
      "ip": ["443"],
      "app": {
        "specht-labs.de/cap/tka": [
          {
            "role": "cluster-admin",
            "period": "8h",
            "priority": 400
          }
        ]
      }
    },
  ]
}
```

## Advanced Patterns

### Time-Based Access

Different roles during business hours vs. emergency access:

```jsonc
{
  "grants": [
    // Business hours: normal access
    {
      "src": ["group:developers"],
      "dst": ["tag:tka"],
      "ip": ["443"],
      "app": {
        "specht-labs.de/cap/tka": [
          {
            "role": "edit",
            "period": "4h",
            "priority": 100
          }
        ]
      }
    },
    // Emergency access: extended permissions
    {
      "src": ["group:oncall"],
      "dst": ["tag:tka"],
      "ip": ["443"],
      "app": {
        "specht-labs.de/cap/tka": [
          {
            "role": "cluster-admin",
            "period": "1h",  // Short window for emergency access
            "priority": 200  // Give it a higher priority, for cases where the oncall engineer is a member of group:developers and group:oncall
          }
        ]
      }
    }
  ]
}
```

### Environment-Specific Access

Use different capability names for different environments:

```jsonc
{
  "tagOwners": {
    "tag:tka-prod": ["group:sre"],
    "tag:tka-dev": ["group:developers"]
  },
  "grants": [
    // Production: limited access, long audit trail
    {
      "src": ["group:sre"],
      "dst": ["tag:tka-prod"],
      "ip": ["443"],
      "app": {
        "specht-labs.de/cap/tka-prod": [
          {
            "role": "admin-readonly",
            "period": "2h",
            "priority": 100
          }
        ]
      }
    },
    // Development: full access for testing
    {
      "src": ["group:developers"],
      "dst": ["tag:tka-dev"],
      "ip": ["443"],
      "app": {
        "specht-labs.de/cap/tka-dev": [
          {
            "role": "cluster-admin",
            "period": "8h",
            "priority": 200
          }
        ]
      }
    }
  ]
}
```

## Configuration Parameters

### Required Fields

- **`src`**: Array of users, groups, or device tags who get access
- **`dst`**: Array containing `tag:tka` (or your custom tag)
- **`ip`**: Array with the port TKA runs on (usually `["443"]`)
- **`app`**: Object with capability name as key

### Capability Object

- **`role`**: Must be a valid Kubernetes ClusterRole name
- **`period`**: Duration string (e.g., `1h`, `30m`, `8h`, `2h30m`)
- **`priority`**: Integer value for rule precedence (higher values take precedence)

### Common Kubernetes Roles

- **`cluster-admin`**: Full cluster access
- **`admin`**: Full access to namespaced resources
- **`edit`**: Read/write access to most resources
- **`view`**: Read-only access to most resources

### Priority System

TKA uses a priority-based system to resolve conflicts when a user matches multiple capability rules. Understanding how priorities work is crucial for designing secure and predictable access control.

#### How Priority Works

1. **Higher Priority Wins**: Rules with higher priority values take precedence over lower priority rules
2. **Unique Priorities Required**: All rules that could apply to the same user must have different priority values
3. **Same Priority = Error**: If multiple matching rules have the same priority, TKA rejects the request with a 400 error

#### Priority Assignment Strategy

- **400+**: Administrative roles (`cluster-admin`, `admin`)
- **200-399**: Elevated/emergency access (`oncall`, `incident-response`)
- **100-199**: Standard access (`edit`, `developer`)
- **1-99**: Read-only access (`view`, `readonly`)

#### Example Priority Hierarchy

```jsonc
{
  "grants": [
    // Read-only access for all engineers
    {
      "src": ["group:engineers"],
      "dst": ["tag:tka"],
      "app": {
        "specht-labs.de/cap/tka": [
          {
            "role": "view",
            "period": "8h",
            "priority": 50  // Lowest priority
          }
        ]
      }
    },
    // Developer access for dev team
    {
      "src": ["group:developers"],
      "dst": ["tag:tka"],
      "app": {
        "specht-labs.de/cap/tka": [
          {
            "role": "edit",
            "period": "4h",
            "priority": 150  // Overrides engineers group
          }
        ]
      }
    },
    // Emergency access for on-call
    {
      "src": ["group:oncall"],
      "dst": ["tag:tka"],
      "app": {
        "specht-labs.de/cap/tka": [
          {
            "role": "cluster-admin",
            "period": "1h",
            "priority": 300  // Highest priority for emergencies
          }
        ]
      }
    }
  ]
}
```

In this example:

- A developer who is also on-call gets `cluster-admin` (priority 300)
- A developer not on-call gets `edit` (priority 150)
- An engineer who isn't a developer gets `view` (priority 50)

#### Common Priority Pitfalls

1. **Same Priority Error**:

   ```jsonc
   // BAD: Both rules have priority 100
   {"src": ["alice"], "app": {"tka": [{"role": "admin", "priority": 100}]}},
   {"src": ["group:admins"], "app": {"tka": [{"role": "edit", "priority": 100}]}}
   ```

2. **Unexpected Priority Ordering**:

   ```jsonc
   // BAD: Lower privilege has higher priority
   {"src": ["group:admins"], "app": {"tka": [{"role": "cluster-admin", "priority": 100}]}},
   {"src": ["group:readonly"], "app": {"tka": [{"role": "view", "priority": 200}]}}
   ```

#### Debugging Priority Issues

When you get a "Multiple capability rules with the same priority found" error:

1. Check all grants that could match your user/groups
2. Ensure each has a unique priority value
3. Use `tka --debug login` to see which rules are being evaluated
4. Review the server logs for detailed rule matching information

## Server Configuration

Ensure your TKA server uses the same capability name:

### Via Command Line

```bash
tka-server serve --cap-name specht-labs.de/cap/tka
```

### Via Configuration File

```yaml
tailscale:
  capName: specht-labs.de/cap/tka
```

### Via Environment Variable

```bash
export TKA_TAILSCALE_CAPNAME=specht-labs.de/cap/tka
```

## Validation and Testing

### Check ACL Syntax

Use the Tailscale admin console ACL editor to validate JSON syntax before saving.

### Test Access

```bash
# Check if you have the expected capability
tka get login

# Verify your role assignment
kubectl auth whoami
kubectl auth can-i "*" "*"  # Test cluster-admin
kubectl auth can-i get pods  # Test basic access
```

### Common Issues

1. **Multiple Rules**: Each user should have only one capability rule

   ```jsonc
   // BAD: Multiple rules for same user
   {
     "grants": [
       {"src": ["alice"], "app": {"tka": [{"role": "admin", "period": "8h", "priority": 200}]}},
       {"src": ["alice"], "app": {"tka": [{"role": "view", "period": "4h", "priority": 100}]}}
     ]
   }

   // GOOD: Single rule per user
   {
     "grants": [
       {"src": ["alice"], "app": {"tka": [{"role": "admin", "period": "8h", "priority": 200}]}}
     ]
   }
   ```

   ::: warning
   TKA **does support priority** in capability grants. Each capability rule must have a unique priority value.
   If multiple rules have the same priority, TKA will reject the request with a 400 error to prevent
   non-deterministic behavior. Always assign different priority values when a user might match multiple rules.
   :::

2. **Role Doesn't Exist**: Ensure the role exists in your cluster

   ```bash
   kubectl get clusterrole your-role-name
   ```

3. **Port Mismatch**: IP field must match TKA server port

   ```bash
   # If server runs on port 8443
   "ip": ["8443"]  # Not 443
   ```

## Security Best Practices

1. **Principle of Least Privilege**: Start with minimal roles and expand as needed
2. **Short Periods**: Use shorter periods for higher privileges
3. **Regular Rotation**: Encourage users to logout and re-authenticate regularly
4. **Audit Trails**: Monitor who accesses what via Kubernetes audit logs
5. **Emergency Access**: Have a separate capability for emergency situations

## Related Documentation

- [Tailscale Capability Grants](https://tailscale.com/kb/1324/grants)
- [Kubernetes RBAC](https://kubernetes.io/docs/reference/access-authn-authz/rbac/)
- [TKA Configuration Reference](../reference/configuration.md)
- [Troubleshooting ACL Issues](./troubleshooting.md#authentication-issues)
