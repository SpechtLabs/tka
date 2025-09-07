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
               "period": "8h"               // Token lifetime
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
    // Full admin access for 8 hours
    {
      "src": ["group:k8s-admins"],
      "dst": ["tag:tka"],
      "ip": ["443"],
      "app": {
        "specht-labs.de/cap/tka": [
          {
            "role": "cluster-admin",
            "period": "8h"
          }
        ]
      }
    },
    // Developer access for 4 hours
    {
      "src": ["group:developers"],
      "dst": ["tag:tka"],
      "ip": ["443"],
      "app": {
        "specht-labs.de/cap/tka": [
          {
            "role": "edit",
            "period": "4h"
          }
        ]
      }
    },
    // Read-only access for 2 hours
    {
      "src": ["group:viewers"],
      "dst": ["tag:tka"],
      "ip": ["443"],
      "app": {
        "specht-labs.de/cap/tka": [
          {
            "role": "view",
            "period": "2h"
          }
        ]
      }
    }
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
            "period": "4h"
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
            "period": "1h"  // Short window for emergency access
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
            "period": "2h"
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
            "period": "8h"
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

### Common Kubernetes Roles

- **`cluster-admin`**: Full cluster access
- **`admin`**: Full access to namespaced resources
- **`edit`**: Read/write access to most resources
- **`view`**: Read-only access to most resources

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
       {"src": ["alice"], "app": {"tka": [{"role": "admin"}]}},
       {"src": ["alice"], "app": {"tka": [{"role": "view"}]}}
     ]
   }

   // GOOD: Single rule per user
   {
     "grants": [
       {"src": ["alice"], "app": {"tka": [{"role": "admin"}]}}
     ]
   }
   ```

   ::: info
   The problem with "multiple rules for the same user" is that there is no priority in the grants,
   nor can we automatically apply least or highest privilege principles as we do not know (programatically)
   which role grants you more or less access. All we know is the RBAC cluster-role name.
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
