// Favorites Page - ProjectParty
import React, { useState, useEffect } from 'react';
import { api } from '../../services/api';

export default function Favorites() {
  const [favorites, setFavorites] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api.getFavorites().then(data => {
      setFavorites(data.favorites || []);
      setLoading(false);
    }).catch(() => setLoading(false));
  }, []);

  const handleRemove = async (id: string) => {
    await api.removeFavorite(id);
    setFavorites(favorites.filter(f => f.id !== id));
  };

  if (loading) return <div>Loading...</div>;

  return (
    <div className="favorites-page">
      <h1>My Favorites</h1>
      {favorites.length === 0 ? (
        <p>No favorites yet.</p>
      ) : (
        <div className="tokens-grid">
          {favorites.map((t: any) => (
            <div key={t.id} className="token-card">
              <img src={t.logo_url} alt={t.name} />
              <h3>{t.name}</h3>
              <p>{t.symbol}</p>
              <button onClick={() => handleRemove(t.id)}>Remove</button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
