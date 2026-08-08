/**
 * TigerWallet Admin - Feature Management Page
 * Complete implementation with backend connectivity
 */

import React, { useState, useEffect } from 'react';
import { useTheme } from '../hooks/useTheme';
import { featuresAPI } from '../services/api';

interface Feature {
  id: string;
  name: string;
  description: string;
  category: string;
  enabled: boolean;
  rolloutPercentage: number;
  createdAt: string;
  updatedAt: string;
}

export const FeaturesPage: React.FC = () => {
  const { isDark, toggleTheme } = useTheme();
  const [features, setFeatures] = useState<Feature[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedCategory, setSelectedCategory] = useState('all');
  const [showAddModal, setShowAddModal] = useState(false);
  const [editingFeature, setEditingFeature] = useState<Feature | null>(null);

  useEffect(() => {
    loadFeatures();
  }, []);

  const loadFeatures = async () => {
    setLoading(true);
    try {
      const response = await featuresAPI.getAll();
      setFeatures(response.data);
    } catch (error) {
      console.error('Failed to load features:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleToggle = async (featureId: string) => {
    try {
      await featuresAPI.toggle(featureId);
      loadFeatures();
    } catch (error) {
      console.error('Failed to toggle feature:', error);
    }
  };

  const handleCreate = async (data: any) => {
    try {
      await featuresAPI.create(data);
      loadFeatures();
      setShowAddModal(false);
    } catch (error) {
      console.error('Failed to create feature:', error);
    }
  };

  const handleUpdate = async (featureId: string, data: any) => {
    try {
      await featuresAPI.update(featureId, data);
      loadFeatures();
      setEditingFeature(null);
    } catch (error) {
      console.error('Failed to update feature:', error);
    }
  };

  const categories = ['all', ...new Set(features.map(f => f.category))];

  const filteredFeatures = selectedCategory === 'all'
    ? features
    : features.filter(f => f.category === selectedCategory);

  return (
    <div className={`page-container ${isDark ? 'dark' : 'light'}`}>
      <div className="page-header">
        <h1>Feature Management</h1>
        <button className="theme-btn" onClick={toggleTheme}>
          {isDark ? '☀️ Light' : '🌙 Dark'}
        </button>
      </div>

      <div className="page-actions">
        <div className="category-tabs">
          {categories.map(category => (
            <button
              key={category}
              className={`tab ${selectedCategory === category ? 'active' : ''}`}
              onClick={() => setSelectedCategory(category)}
            >
              {category === 'all' ? 'All' : category}
            </button>
          ))}
        </div>
        <button className="btn-primary" onClick={() => setShowAddModal(true)}>
          + Add Feature
        </button>
      </div>

      {loading ? (
        <div className="loading">Loading features...</div>
      ) : (
        <div className="features-list">
          {filteredFeatures.map(feature => (
            <div key={feature.id} className="feature-card">
              <div className="feature-header">
                <div className="feature-info">
                  <h3>{feature.name}</h3>
                  <p>{feature.description}</p>
                </div>
                <div className="feature-toggle">
                  <label className="switch">
                    <input
                      type="checkbox"
                      checked={feature.enabled}
                      onChange={() => handleToggle(feature.id)}
                    />
                    <span className="slider"></span>
                  </label>
                </div>
              </div>
              <div className="feature-details">
                <span className="category-badge">{feature.category}</span>
                <div className="rollout">
                  <span>Rollout: {feature.rolloutPercentage}%</span>
                  <div className="progress-bar">
                    <div
                      className="progress"
                      style={{ width: `${feature.rolloutPercentage}%` }}
                    ></div>
                  </div>
                </div>
                <button
                  className="btn-secondary"
                  onClick={() => setEditingFeature(feature)}
                >
                  Edit
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {showAddModal && (
        <div className="modal-overlay" onClick={() => setShowAddModal(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Add New Feature</h2>
            <button className="close-btn" onClick={() => setShowAddModal(false)}>×</button>
            <form onSubmit={(e) => {
              e.preventDefault();
              const form = e.target as HTMLFormElement;
              handleCreate({
                name: (form.elements.namedItem('name') as HTMLInputElement).value,
                description: (form.elements.namedItem('description') as HTMLInputElement).value,
                category: (form.elements.namedItem('category') as HTMLInputElement).value,
                rolloutPercentage: parseInt((form.elements.namedItem('rollout') as HTMLInputElement).value),
              });
            }}>
              <div className="form-group">
                <label>Feature Name</label>
                <input type="text" name="name" required placeholder="Feature name" />
              </div>
              <div className="form-group">
                <label>Description</label>
                <textarea name="description" required placeholder="Feature description"></textarea>
              </div>
              <div className="form-group">
                <label>Category</label>
                <input type="text" name="category" required placeholder="e.g., Trading, Payments" />
              </div>
              <div className="form-group">
                <label>Rollout Percentage (0-100)</label>
                <input type="number" name="rollout" min="0" max="100" defaultValue="0" />
              </div>
              <button type="submit" className="btn-primary">Create Feature</button>
            </form>
          </div>
        </div>
      )}

      {editingFeature && (
        <div className="modal-overlay" onClick={() => setEditingFeature(null)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Edit Feature</h2>
            <button className="close-btn" onClick={() => setEditingFeature(null)}>×</button>
            <form onSubmit={(e) => {
              e.preventDefault();
              const form = e.target as HTMLFormElement;
              handleUpdate(editingFeature.id, {
                name: (form.elements.namedItem('name') as HTMLInputElement).value,
                description: (form.elements.namedItem('description') as HTMLInputElement).value,
                rolloutPercentage: parseInt((form.elements.namedItem('rollout') as HTMLInputElement).value),
              });
            }}>
              <div className="form-group">
                <label>Feature Name</label>
                <input type="text" name="name" defaultValue={editingFeature.name} required />
              </div>
              <div className="form-group">
                <label>Description</label>
                <textarea name="description" defaultValue={editingFeature.description} required />
              </div>
              <div className="form-group">
                <label>Rollout Percentage</label>
                <input type="number" name="rollout" min="0" max="100" defaultValue={editingFeature.rolloutPercentage} />
              </div>
              <button type="submit" className="btn-primary">Save Changes</button>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};

export default FeaturesPage;
